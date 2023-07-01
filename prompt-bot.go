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

				switch eventsAPIEvent.Type {
				case slackevents.CallbackEvent:
					innerEvent := eventsAPIEvent.InnerEvent
					switch event := innerEvent.Data.(type) {
					case *slackevents.MessageEvent:
						if event.User != *_BotID {
							debugLog("text: " + event.Text)

							actualAttachmentJson, err := json.Marshal(event.Files)
							if err != nil {
								fmt.Println("expected no error unmarshaling attachment with blocks, got: %v", err)
							}
							mess := string(actualAttachmentJson)

							str, eflag := validMessage(event.Text, *_Record, *_Result, *_Search, mess, *_Ini)
							switch eflag {
							case 0:
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
								strc := rejectEscape(event.Text)
								entryID := RandStr(8)
								debugLog("prompt entry: " + event.Username + " prompt id: " + entryID)
								writePicIni(api, strc, strings.Replace(event.Text, "\n", "", 1), str, Dir+entryID, *_Ini)

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
							case 10:
								PostMessage(api, event.Channel, "Please specify search words")
							case 20:
								//PostMessage(api, event.Channel, "Please specify prompt words")
							case 30:
								PostMessage(api, event.Channel, "That prompt is already registered.")
							case 40:
								PostMessage(api, event.Channel, "no index exits!")
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

func validMessage(text, record, result, search, mess, inifile string) (string, int) {
	fmt.Println(text)
	if strings.Index(text, search+" ") == 0 {
		_, count := readText(inifile, false)
		if count == 0 {
			return "", 40
		}
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
		if checkSamePromt(stra, inifile) == true {
			return "", 30
		}
		if strings.Index(stra, result+"\n") != -1 {
			return stra, 2
		}
	}

	if strings.Index("このメッセージにはインタラクティブ要素が含まれます", text) == -1 {
		//"event":{"type":"message","text":"\u3053\u306e\u30e1\u30c3\u30bb\u30fc\u30b8\u306b\u306f\u30a4\u30f3\u30bf\u30e9\u30af\u30c6\u30a3\u30d6\u8981\u7d20\u304c\u542b\u307e\u308c\u307e\u3059\u3002"
		//"files":[{"id":"F05FV4XKWQY","created":1688210619,"timestamp":1688210619,"name":"image.png","title":"image.png","mimetype":"image\/png","filetype":"png","pretty_type":"PNG","user":"U024ZT3BHU5","user_team":"T024W6FDUKG","editable":false,"size":1016,"mode":"hosted","is_external":false,"external_type":"","is_public":true,"public_url_shared":false,"display_as_bot":false,"username":"","url_private":"https:\/\/files.slack.com\/files-pri\/T024W6FDUKG-F05FV4XKWQY\/image.png","url_private_download":"https:\/\/files.slack.com\/files-pri\/T024W6FDUKG-F05FV4XKWQY\/download\/image.png","media_display_type":"unknown","thumb_64":"https:\/\/files.slack.com\/files-tmb\/T024W6FDUKG-F05FV4XKWQY-f4fedd55b4\/image_64.png","thumb_80":"https:\/\/files.slack.com\/files-tmb\/T024W6FDUKG-F05FV4XKWQY-f4fedd55b4\/image_80.png","thumb_360":"https:\/\/files.slack.com\/files-tmb\/T024W6FDUKG-F05FV4XKWQY-f4fedd55b4\/image_360.png","thumb_360_w":134,"thumb_360_h":53,"thumb_160":"https:\/\/files.slack.com\/files-tmb\/T024W6FDUKG-F05FV4XKWQY-f4fedd55b4\/image_160.png","original_w":134,"original_h":53,"thumb_tiny":"AwASADDRkyUIBI+lMG\/advHP8XpinyDcuOfwqMo5jYKdhJ4xQAZlx1HtSgy5Odp44pqxyhyTISMdM07bIBw3PvzQA5S+fm249qfkUi5A+Y5NLmgAooooAKKKKACiiigD\/9k=","permalink":"https:\/\/5-iab3526.slack.com\/files\/U024ZT3BHU5\/F05FV4XKWQY\/image.png","permalink_public":"https:\/\/slack-files.com\/T024W6FDUKG-F05FV4XKWQY-c14409118a","has_rich_preview":false,"file_access":"visible"}],"upload":false,"user":"U024ZT3BHU5","display_as_bot":false,"ts":"1688210623.400079","blocks":[{"type":"rich_text","block_id":"JX5er",
		//"elements":[{"type":"rich_text_section","elements":[{"type":"text","text":"record\nExplain antibiotics\nA:"}]}]}],"client_msg_id":"8c1c8ca4-2f31-42e3-85a4-42a48698ce01","channel":"C0252VAF0N6","subtype":"file_share","event_ts":"1688210623.400079","channel_type":"channel"},"type":"event_callback","event_id":"Ev05EQT3K1P1","event_time":1688210623,"authorizations":[{"enterprise_id":null,"team_id":"T024W6FDUKG","user_id":"U026G2JFYC9","is_bot":true,"is_enterprise_install":false}],"is_ext_shared_channel":false,"event_context":"4-eyJldCI6Im1lc3NhZ2UiLCJ0aWQiOiJUMDI0VzZGRFVLRyIsImFpZCI6IkEwMjVXS0xOR0xFIiwiY2lkIjoiQzAyNTJWQUYwTjYifQ"},"type":"events_api","accepts_response_payload":false,"retry_attempt":0,"retry_reason":""}
		fmt.Println(mess)
		stra := strings.Replace(text, record, "", -1)
		if len(stra) < 1 {
			return "", 20
		}
		if checkSamePromt(stra, inifile) == true {
			return "", 30
		}
		if strings.Index(stra, result+"\n") != -1 {
			return stra, 2
		}
		if strings.Index(mess, "url_private_download") != -1 {
			strb := strings.Split(mess, "url_private_download")
			strc := strings.Split(strb[1], ",")
			strd := strings.Replace(strc[0], "\"", "", -1)
			strd = strings.Replace(strd, "\\", "", -1)
			strd = strings.Replace(strd, ":", "", 1)
			return strd, 1
		}

	}

	return "", -1
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
