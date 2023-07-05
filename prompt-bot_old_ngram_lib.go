package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lestrrat-go/ngram"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

var (
	debug      bool
	logging    bool
	rs1Letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	_Debug := flag.Bool("debug", false, "[-debug=debug mode (true is enable)]")
	_Logging := flag.Bool("log", false, "[-log=logging mode (true is enable)]")
	_Record := flag.String("record", "record", "[-record=These are the words used to register the prompt]")
	_Result := flag.String("result", "result", "[-result=A word that specifies the output of the prompt]")
	_Search := flag.String("search", "search", "[-search=The word when searching for prompts.]")
	_Ini := flag.String("ini", "prompt-bot.ini", "[-ini=config file name.")
	_Dir := flag.String("dir", "data", "[-dir=Directory to store registered information.")
	_BotID := flag.String("botid", "U026G2JFYC9", "[-botid=Define IDs for bots to prevent response loops.")

	flag.Parse()

	debug = bool(*_Debug)
	logging = bool(*_Logging)

	var index *ngram.Index
	_, count := readText(*_Ini, false)
	if count > 0 {
		index = ngram.NewIndex(count)
		index = reLoad(*_Ini, index)
	}

	Dir := ""
	prevDir, _ := filepath.Abs(".")
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		Dir = prevDir + "/" + *_Dir + "/"
	} else {
		Dir = prevDir + "\\" + *_Dir + "\\"
	}
	debugLog("set dir: " + Dir)

	if _, err := os.Stat(Dir); os.IsNotExist(err) {
		os.Mkdir(Dir, 0777)
	}

	file, err := os.OpenFile(*_Ini, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		fmt.Fprintf(os.Stderr, "SLACK_APP_TOKEN must be set.\n")
		os.Exit(1)
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		fmt.Fprintf(os.Stderr, "SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		fmt.Fprintf(os.Stderr, "SLACK_BOT_TOKEN must be set.\n")
		os.Exit(1)
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		fmt.Fprintf(os.Stderr, "SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	api := slack.New(
		botToken,
		slack.OptionDebug(debug),
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
		socketmode.OptionDebug(debug),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				fmt.Println("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				fmt.Println("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				fmt.Println("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					fmt.Printf("Ignored %+v\n", evt)
					continue
				}

				client.Ack(*evt.Request)
				actualAttachmentJson, err := json.Marshal(*evt.Request)
				if err != nil {
					fmt.Println("expected no error unmarshaling attachment with blocks, got: %v", err)
				}
				mess := string(actualAttachmentJson)

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent

					switch event := innerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						if event.User != *_BotID {
							debugLog("text: " + event.Text)

							str, strr, eflag := validMessage(event.Text, *_Record, *_Result, *_Search, mess, *_Ini)
							switch eflag {
							case 0:
								debugLog("search word: " + str)
								matches := index.FindBestMatch(str)
								debugLog("matched: " + matches)
								strb := strings.Split(matches, "\t")
								if strb[3] == "t" {
									debugLog("prompt serch: " + strb[1])
									strc, _ := readText(strb[1], true)
									PostMessage(api, event.Channel, "prompt\n```\n"+strc+"```\n")
									strc, _ = readText(strb[2], true)
									PostMessage(api, event.Channel, "result\n```\n"+strc+"```\n")
								} else {
									debugLog("prompt serch: " + strb[1])
									strc, _ := readText(strb[1], true)
									PostMessage(api, event.Channel, "prompt\n```\n"+strc+"```\n")
									params := slack.FileUploadParameters{
										Title:    "result",
										File:     strb[2],
										Filetype: "binary",
										Channels: []string{event.Channel},
									}
									file, err := api.UploadFile(params)
									if err != nil {
										fmt.Printf("upload error: %s\n", err)
									}
									fmt.Printf("upload! Name: %s, URL: %s\n", file.Name, file.URL, file.ID)
								}
							case 1:
								strc := rejectEscape(str)
								entryID := RandStr(8)
								debugLog("prompt entry: " + event.Username + " prompt id: " + entryID)
								writePicIni(api, strings.Replace(strc, "\n", "", 1), strings.Replace(str, "\n", "", 1), strr, Dir+entryID, *_Ini)

								_, count := readText(*_Ini, false)
								index = ngram.NewIndex(count)
								index = reLoad(*_Ini, index)

								PostMessage(api, event.Channel, "Text & Picture Registered!")
							case 2:
								strb := strings.Split(str, *_Result)
								strc := rejectEscape(strb[0])
								entryID := RandStr(8)
								debugLog("prompt entry: " + event.Username + " prompt id: " + entryID)
								writeTextIni(strc, strings.Replace(strb[0], "\n", "", 1), strings.Replace(strb[1], "\n", "", 1), Dir+entryID, *_Ini)

								_, count := readText(*_Ini, false)
								index = ngram.NewIndex(count)
								index = reLoad(*_Ini, index)

								PostMessage(api, event.Channel, "Text Source Registered!")
							case -1:
								if len(str) > 0 {
									PostMessage(api, event.Channel, str)
								}
							}
						}
					}
				default:
					client.Debugf("unsupported Events API event received")
				}
			default:
				fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()
	client.Run()

	os.Exit(0)
}

func rejectEscape(str string) string {
	stra := strings.Replace(str, " ", "", -1)
	stra = strings.Replace(stra, "　", "", -1)
	stra = strings.Replace(str, "\n", "", -1)
	stra = strings.Replace(stra, "\t", "", -1)
	return stra
}

func PostMessage(api *slack.Client, channel, message string) {
	_, _, err := api.PostMessage(channel, slack.MsgOptionText(message, false))
	if err != nil {
		fmt.Printf("failed posting message: %v", err)
	}

}

func writePicIni(api *slack.Client, indexWord, prompt, url, filename, indexFile string) {
	writeFile(filename+"_prompt", prompt)

	f, err := os.Create(filename + "_result")
	defer f.Close()
	if err != nil {
		return
	}
	err = api.GetFile(url, f)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	file, err := os.OpenFile(indexFile, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	fmt.Fprintln(file, indexWord+"\t"+filename+"_prompt"+"\t"+filename+"_result"+"\t"+"p")
}

func writeTextIni(indexWord, prompt, result, filename, indexfile string) {
	writeFile(filename+"_prompt", prompt)
	writeFile(filename+"_result", result)

	file, err := os.OpenFile(indexfile, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	fmt.Fprintln(file, indexWord+"\t"+filename+"_prompt"+"\t"+filename+"_result"+"\t"+"t")
}

func writeFile(filename, stra string) bool {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer file.Close()

	_, err = file.WriteString(stra)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func validMessage(text, record, result, search, mess, inifile string) (string, string, int) {
	if strings.Index(text, search+" ") == 0 {
		_, count := readText(inifile, false)
		if count == 0 {
			return "no index exits!", "", -1
		}
		stra := strings.Replace(text, search, "", -1)
		if len(stra) < 1 {
			return "Please specify search words", "", -1
		}
		return stra, "", 0
	}

	if strings.Index(text, record) == 0 {
		stra := strings.Replace(text, record, "", -1)
		if len(stra) < 1 {
			return "Please specify prompt words", "", -1
		}
		if checkSamePromt(stra, inifile) == true {
			return "That prompt is already registered", "", -1
		}
		if strings.Index(stra, result+"\n") != -1 {
			return stra, "", 2
		}
	}

	if strings.Index("url_private_download", mess) == -1 && strings.Index("rich_text_section", mess) == -1 {
		strb := strings.Split(mess, "url_private_download")
		strc := strings.Split(strb[1], ",")
		strd := strings.Replace(strc[0], "\"", "", -1)
		strd = strings.Replace(strd, "\\", "", -1)
		strd = strings.Replace(strd, ":", "", 1)
		debugLog("file url: " + strd)

		stre := strings.Split(mess, "rich_text_section")
		strf := strings.Split(stre[1], "text")
		strg := strings.Replace(strf[2], "\":", "", -1)
		strh := strings.Replace(strings.Split(strg, "}")[0], "\"", "", -1)
		strh = strings.Replace(strh, "\\n", "", -1)

		stra := strings.Replace(strh, record, "", -1)
		debugLog("rich Text: " + stra)
		if len(stra) < 1 {
			return "Please specify prompt words", "", -1
		}
		if checkSamePromt(stra, inifile) == true {
			return "That prompt is already registered", "", -1
		}
		return stra, strd, 1
	}

	return "", "", -1
}

func checkSamePromt(prompt, inifile string) bool {
	str := rejectEscape(prompt)
	fmt.Println(str)
	f, err := os.Open(inifile)
	if err != nil {
		fmt.Printf("os.Open: %#v\n", err)
		os.Exit(-1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		strb := scanner.Text()
		strc := strings.Split(strb, "\t")
		if strings.Index(str, strc[0]) == 0 {
			fmt.Println(str)
			fmt.Println(strc[0])
			return true
		}
		fmt.Println(strc[0])
	}

	if err = scanner.Err(); err != nil {
		fmt.Printf("scanner.Err: %#v\n", err)
		os.Exit(-1)
	}

	return false
}

func reLoad(filename string, index *ngram.Index) *ngram.Index {
	fp, err := os.Open(filename)
	if err != nil {
		fmt.Printf("os.Open: %#v\n", err)
		os.Exit(-1)
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)

	for scanner.Scan() {
		str := scanner.Text()
		index.AddString(str)
		debugLog("add Index: " + str)
	}

	if err = scanner.Err(); err != nil {
		fmt.Printf("scanner.Err: %#v\n", err)
		os.Exit(-1)
	}
	return index
}

func readText(filename string, sFlag bool) (string, int) {
	str := ""
	line := 0

	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("os.Open: %#v\n", err)
		return "", 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if sFlag == true {
			str = str + scanner.Text() + "\n"
		}
		line++
	}

	if err = scanner.Err(); err != nil {
		fmt.Printf("scanner.Err: %#v\n", err)
		os.Exit(-1)
	}

	fmt.Printf("lines: %d\n", line)
	return str, line
}

func RandStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = rs1Letters[rand.Intn(len(rs1Letters))]
	}
	return string(b)
}

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func debugLog(message string) {
	var file *os.File
	var err error

	if debug == true {
		fmt.Println(message)
	}

	if logging == false {
		return
	}

	const layout = "2006-01-02_15"
	const layout2 = "2006/01/02 15:04:05"
	t := time.Now()
	filename := t.Format(layout) + ".log"
	logHead := "[" + t.Format(layout2) + "] "

	if Exists(filename) == true {
		file, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, 0666)
	} else {
		file, err = os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	}

	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()
	fmt.Fprintln(file, logHead+message)
}
