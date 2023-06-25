package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
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
	_Record := flag.String("record", "record", "[-record=These are the words used to register the prompt")
	_Result := flag.String("result", "result", "[-result=A word that specifies the output of the prompt")
	_Search := flag.String("search", "search", "[-search=The word when searching for prompts.")
	_Ini := flag.String("ini", "prompt-bot.ini", "[-ini=config file name.")

	flag.Parse()

	debug = bool(*_Debug)
	logging = bool(*_Logging)

	_, count := readText(*_Ini, false)
	index := ngram.NewIndex(count)
	index = reLoad(*_Ini, index)

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

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch event := innerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						fmt.Println("text: " + event.Text)

						actualAttachmentJson, err := json.Marshal(event.Files)
						if err != nil {
							fmt.Println("expected no error unmarshaling attachment with blocks, got: %v", err)
						}
						mess := string(actualAttachmentJson)

						str, eflag := validMessage(event.Text, *_Record, *_Result, *_Search, mess)
						switch eflag {
						case 0:
							matches := index.FindBestMatch(str)
							fmt.Printf("matched %s\n", matches)
							strb := strings.Split(matches, "\t")
							if strb[3] == "t" {
								strc, _ := readText(strb[1], true)
								_, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("prompt\n```\n"+strc+"```\n", false))
								if err != nil {
									fmt.Printf("failed posting message: %v", err)
								}
								strc, _ = readText(strb[2], true)
								_, _, err = api.PostMessage(event.Channel, slack.MsgOptionText("result\n```\n"+strc+"```\n", false))
								if err != nil {
									fmt.Printf("failed posting message: %v", err)
								}
							} else {
								strc, _ := readText(strb[1], true)
								_, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("prompt\n```\n"+strc+"```\n", false))
								if err != nil {
									fmt.Printf("failed posting message: %v", err)
								}
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
							strc := rejectEscape(event.Text)
							writePicIni(api, strc, event.Text, str, RandStr(8), *_Ini)
							index = reLoad(*_Ini, index)
							_, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("Registered!", false))
							if err != nil {
								fmt.Printf("failed posting message: %v", err)
							}
						case 2:
							strb := strings.Split(str, *_Result)
							strc := rejectEscape(strb[0])
							writeTextIni(strc, strb[0], strb[0], RandStr(8), *_Ini)
							index = reLoad(*_Ini, index)
							_, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("Registered!", false))
							if err != nil {
								fmt.Printf("failed posting message: %v", err)
							}
						case 10:
							_, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("Please specify search words", false))
							if err != nil {
								fmt.Printf("failed posting message: %v", err)
							}
						case 20:
							_, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("Please specify prompt words", false))
							if err != nil {
								fmt.Printf("failed posting message: %v", err)
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
	stra := strings.Replace(str, "\n", "", -1)
	stra = strings.Replace(stra, "\t", "", -1)
	stra = strings.Replace(stra, " ", "", -1)
	stra = strings.Replace(stra, "　", "", -1)
	return stra
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

	_, err = file.WriteString(stra + "\n")
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func validMessage(text, record, result, search, mess string) (string, int) {
	if strings.Index(text, search) == 0 {
		stra := strings.Replace(text, search, "", -1)
		if len(stra) < 1 {
			return "", 10
		}
		return stra, 0
	}

	if strings.Index(text, record) == 0 {
		stra := strings.Replace(text, record, "", -1)
		if len(stra) < 1 {
			return "", 20
		}
		if strings.Index(stra, result+"\n") != -1 {
			return stra, 2
		}
		if strings.Index(mess, "url_private_download") != -1 {
			strb := strings.Split(mess, "url_private_download")
			fmt.Println(strb)
			strc := strings.Split(strb[1], ",")
			fmt.Println(strc[0])
			strd := strings.Replace(strc[0], "\"", "", -1)
			strd = strings.Replace(strd, "\\", "", -1)
			strd = strings.Replace(strd, ":", "", 1)
			fmt.Println(strd)
			return strd, 1
		}
	}

	return "", -1
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
		fmt.Println("add Index: " + str)
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
		os.Exit(-1)
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
