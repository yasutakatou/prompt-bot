package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Lazin/go-ngram"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type promptAction struct {
	ID     string
	USER   string
	ACTION string
}

var (
	debug      bool
	logging    bool
	rs1Letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	actions    []promptAction
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	_Debug := flag.Bool("debug", false, "[-debug=debug mode (true is enable)]")
	_Logging := flag.Bool("log", false, "[-log=logging mode (true is enable)]")
	_Record := flag.String("record", "record", "[-record=These are the words used to register the prompt]")
	_Result := flag.String("result", "result", "[-result=A word that specifies the output of the prompt]")
	_threshold := flag.String("threshold", "0.2", "[-threshold=Threshold for best matching sentences.]")
	_Ini := flag.String("ini", "prompt-bot.ini", "[-ini=config file name.")
	_Dir := flag.String("dir", "data", "[-dir=Directory to store registered information.")
	_BotID := flag.String("botid", "U026G2JFYC9", "[-botid=Define IDs for bots to prevent response loops.")
	// Search Type
	_like := flag.String("like", "like", "[-like=The word when searching for prompts.]")
	_like3 := flag.String("like3", "like3", "[-like3=The word when searching for prompts.]")
	_match := flag.String("match", "match", "[-match=The word when searching for prompts.]")
	_match3 := flag.String("match3", "match3", "[-match3=The word when searching for prompts.]")
	_Gramsize := flag.Int("Gramsize", 3, "[-Gramsize=N (gram size) to NGramIndex c-tor.]")
	_top := flag.Int("top", 3, "[-top=Change to a number other than TOP 3.]")
	_loop := flag.Int("loop", 24, "[-loop=The interval at which periodic usage reports are output.]")
	_noreport := flag.Bool("noreport", false, "[-debug=Put it in a mode that does not output periodic usage reports.]")
	_reportChannel := flag.String("reportChannel", "XXXXXXXX", "[-reportChannel=Specify which channels to output periodic usage reports.]")
	_historySize := flag.Int("historySize", 100, "[-historySize=Specify the number of statements to look back on channel usage.]")

	flag.Parse()

	debug = bool(*_Debug)
	logging = bool(*_Logging)

	thre, _ := strconv.ParseFloat(*_threshold, 64)

	var index *ngram.NGramIndex
	_, count := readText(*_Ini, false, false)
	if count > 0 {
		index, _ = ngram.NewNGramIndex(ngram.SetN(*_Gramsize))
		index = reLoad(*_Ini, count, *_Gramsize)
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

							astr, astrr, eflag := validMessage(event.Text, *_Record, *_Result, *_like, *_like3, *_match, *_match3, mess, *_Ini)
							str := string(astr)
							strr := string(astrr)

							switch eflag {
							case 10:
								debugLog("like search word: " + str)
								matches, err := index.BestMatch(str, thre)
								if err != nil {
									fmt.Println(err)
									PostMessage(api, event.Channel, "no hit!")
								} else {
									strc, err := index.GetString(matches.TokenID)
									if err != nil {
										fmt.Println(err)
									} else {
										debugLog("matched: " + strc)
										answerSwitch(api, strc, event.Channel, Dir)
									}
								}
							case 11:
								debugLog("like3 search word: " + str)
								matches, err := index.Search(str, thre)
								if err != nil {
									fmt.Println(err)
								} else {
									cnt := len(matches)
									if cnt >= *_top {
										cnt = *_top
									}
									if cnt > 0 {
										for i := 0; i < cnt; i++ {
											s := strconv.Itoa(i + 1)
											strc, err := index.GetString(matches[i].TokenID)
											if err != nil {
												fmt.Println(err)
											} else {
												debugLog(s + " matched: " + strc)
												PostMessage(api, event.Channel, "answer ["+s+"]")
												answerSwitch(api, strc, event.Channel, Dir)
											}
										}
									} else {
										PostMessage(api, event.Channel, "no hit!")
									}
								}
							case 12:
								debugLog("match search word: " + str)
								strs := matchSearch(*_Ini, str)
								if len(strs) > 0 {
									debugLog("matched: " + strs[0])
									answerSwitch(api, strs[0], event.Channel, Dir)
								} else {
									PostMessage(api, event.Channel, "no hit!")
								}
							case 13:
								debugLog("match3 search word: " + str)
								strs := matchSearch(*_Ini, str)
								if err != nil {
									fmt.Println(err)
								} else {
									cnt := len(strs)
									if cnt >= *_top {
										cnt = *_top
									}
									if cnt > 0 {
										for i := 0; i < cnt; i++ {
											s := strconv.Itoa(i + 1)
											if err != nil {
												fmt.Println(err)
											} else {
												debugLog(s + " matched: " + strs[i])
												PostMessage(api, event.Channel, "answer ["+s+"]")
												answerSwitch(api, strs[i], event.Channel, Dir)
											}
										}
									} else {
										PostMessage(api, event.Channel, "no hit!")
									}
								}
							case 1:
								strc := rejectEscape(str)
								entryID := RandStr(8)
								debugLog("prompt entry: " + event.Username + " prompt id: " + entryID)
								writePicIni(api, entryID, strings.Replace(strc, "\n", "", 1), strings.Replace(str, "\n", "", 1), strr, Dir+entryID, *_Ini)

								_, count := readText(*_Ini, false, false)
								index, _ = ngram.NewNGramIndex(ngram.SetN(*_Gramsize))
								index = reLoad(*_Ini, count, *_Gramsize)

								PostMessage(api, event.Channel, "Text & Picture Registered!")
							case 2:
								strb := strings.Split(str, *_Result)
								strc := rejectEscape(strb[0])
								entryID := RandStr(8)
								debugLog("prompt entry: " + event.Username + " prompt id: " + entryID)
								writeTextIni(entryID, strc, strings.Replace(strb[0], "\n", "", 1), strings.Replace(strb[1], "\n", "", 1), Dir+entryID, *_Ini)

								_, count := readText(*_Ini, false, false)
								index, _ = ngram.NewNGramIndex(ngram.SetN(*_Gramsize))
								index = reLoad(*_Ini, count, *_Gramsize)

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
	go client.Run()

	var timeLock float64
	var nowStr string

	for {
		if *_noreport == false {
			const layout = "15:04:05"
			t := time.Now()
			nowTime := strings.Split(t.Format(layout), ":")[0]
			debugLog("nowTime: " + nowTime)

			params := slack.GetConversationHistoryParameters{ChannelID: *_reportChannel, Limit: *_historySize}
			messages, err := api.GetConversationHistory(&params)
			if err != nil {
				fmt.Printf("incident not get: %s\n", err)
				return
			}

			tFlag := false
			rFlag := false
			if Exists("prompt-bot.lock") {
				strs, _ := readText("prompt-bot.lock", true, true)
				tFlag = true
				timeLock, _ = strconv.ParseFloat(strs, 64)
			}

			for _, message := range messages.Messages {
				nowLock, _ := strconv.ParseFloat(message.Timestamp, 64)
				if tFlag == true {
					if nowLock < timeLock {
						break
					}
				}
				if rFlag == false {
					nowStr = message.Timestamp
					rFlag = true
				}
				if strings.Index(message.Text, "prompt ") == 0 {
					strs := strings.Split(message.Text, "```")
					strb := strings.Replace(strs[0], "\n", "", -1)
					strb = strings.Replace(strb, "prompt ", "", -1)
					debugLog("prompt [" + strb + "]")
					checkReaction(api, message.Reactions, strb)
				}
			}
			writeFile("prompt-bot.lock", nowStr, true)
			if len(actions) > 0 {
				portActionReport(api, *_reportChannel)
			}

			time.Sleep(time.Hour * time.Duration(*_loop))
		}
	}
	os.Exit(0)
}

func portActionReport(api *slack.Client, channel string) {
	debugLog("reporting: " + channel)
	strs := "```\n"
	for i := 0; i < len(actions); i++ {
		strs = strs + actions[i].ID + "," + actions[i].USER + "," + actions[i].ACTION + "\n"
	}
	strs = strs + "```\n"
	PostMessage(api, channel, strs)
	actions = nil
}

func multiWordSerch(str, words string) bool {
	var strb []string
	if strings.Index(words, " ") != -1 {
		strb = strings.Split(words, " ")
	} else if strings.Index(words, "　") != -1 {
		strb = strings.Split(words, "　")
	} else {
		strb = append(strb, words)
	}

	wFlag := true
	for i := 0; i < len(strb); i++ {
		if strings.Index(str, strb[i]) == -1 {
			wFlag = false
			break
		}
	}
	return wFlag
}

func matchSearch(filename, word string) []string {
	buff := readTextArray(filename)
	var result []string

	rand.Seed(time.Now().Unix())
	alls := len(buff)
	rnd := rand.Intn(len(buff))
	if rnd >= (alls / 2) {
		result = uploop(rnd, alls, buff, word)
	} else {
		result = downloop(rnd, alls, buff, word)
	}
	return result
}

func uploop(rnd, alls int, buff []string, word string) []string {
	var result []string

	cnt := alls
	for i := rnd; i < len(buff); i++ {
		if multiWordSerch(buff[i], word) == true {
			result = append(result, buff[i])
		}
		cnt = cnt - 1
	}

	for i := 0; i < cnt; i++ {
		if multiWordSerch(buff[i], word) == true {
			result = append(result, buff[i])
		}
	}
	return result
}

func downloop(rnd, alls int, buff []string, word string) []string {
	var result []string

	i := rnd
	if rnd > 0 {
		for {
			if multiWordSerch(buff[i], word) == true {
				result = append(result, buff[i])
			}
			i = i - 1
			if i == -1 {
				break
			}
		}
	}

	count := alls - rnd - 1
	for i := 0; i < count; i++ {
		if multiWordSerch(buff[alls-i-1], word) == true {
			result = append(result, buff[alls-i-1])
		}
	}
	return result
}

func readTextArray(filename string) []string {
	var buff []string

	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("os.Open: %#v\n", err)
		return buff
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		strb := scanner.Text()
		if len(strb) > 1 {
			buff = append(buff, strb)
		}
	}

	if err = scanner.Err(); err != nil {
		fmt.Printf("scanner.Err: %#v\n", err)
		os.Exit(-1)
	}

	return buff
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

func answerSwitch(api *slack.Client, strc, channelID, Dir string) {
	strb := strings.Split(strc, "\t")

	if checkFileTypeText(Dir+strb[1]+"_result") == true {
		debugLog("[result text] prompt serch: " + strb[1])
		strc, _ := readText(Dir+strb[1]+"_prompt", true, false)
		PostMessage(api, channelID, "prompt "+strb[1]+"\n```\n"+strc+"```\n")
		strc, _ = readText(Dir+strb[1]+"_result", true, false)
		PostMessage(api, channelID, "result\n```\n"+strc+"```\n")
	} else {
		debugLog("[result picture] prompt serch: " + strb[1])
		strc, _ := readText(Dir+strb[1]+"_prompt", true, false)
		PostMessage(api, channelID, "prompt "+strb[1]+"\n```\n"+strc+"```\n")
		params := slack.FileUploadParameters{
			Title:    "result",
			File:     Dir + strb[1] + "_result",
			Filetype: "binary",
			Channels: []string{channelID},
		}
		file, err := api.UploadFile(params)
		if err != nil {
			fmt.Printf("upload error: %s\n", err)
		}
		fmt.Printf("upload! Name: %s, URL: %s\n", file.Name, file.URL, file.ID)
	}
}

func checkFileTypeText(filename string) bool {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	mimeType := http.DetectContentType(bytes)
	debugLog(filename + " : is  " + string(mimeType))

	if strings.Index(string(mimeType), "text") != -1 {
		return true
	}
	return false
}

func writePicIni(api *slack.Client, entryID, indexWord, prompt, url, filename, indexFile string) {
	writeFile(filename+"_prompt", prompt, false)

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
	str := fmt.Sprintf("%s", indexWord)
	fmt.Fprintln(file, str+"\t"+entryID)
}

func writeTextIni(entryID, indexWord, prompt, result, filename, indexfile string) {
	writeFile(filename+"_prompt", prompt, false)
	writeFile(filename+"_result", result, false)

	file, err := os.OpenFile(indexfile, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	str := fmt.Sprintf("%s", indexWord)
	fmt.Fprintln(file, str+"\t"+entryID)
}

func writeFile(filename, stra string, overwrite bool) bool {
	var file *os.File
	var err error

	if overwrite == false {
		file, err = os.Create(filename)
	} else {
		file, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	}

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

func validMessage(text, record, result, like, like3, match, match3, mess, inifile string) (string, string, int) {
	text = strings.Replace(text, record, "", -1)

	if strings.Index(mess, "url_private_download") != -1 && strings.Index(mess, "rich_text_section") != -1 && len(text) > 1 {
		strb := strings.Split(mess, "url_private_download")
		strc := strings.Split(strb[1], ",")
		strd := strings.Replace(strc[0], "\"", "", -1)
		strd = strings.Replace(strd, "\\", "", -1)
		strd = strings.Replace(strd, ":", "", 1)
		debugLog("file url: " + strd)

		// stre := strings.Split(mess, "rich_text_section")
		// strf := strings.Split(stre[1], "text")
		// strg := strings.Replace(strf[2], "\":", "", -1)
		// strh := strings.Replace(strings.Split(strg, "}")[0], "\"", "", -1)
		// strh = strings.Replace(strh, "\\n", "", -1)

		// stra := strings.Replace(strh, record, "", -1)
		// debugLog("rich Text: " + string(stra))
		if len(text) < 1 {
			return "Please specify prompt words", "", -1
		}
		if checkSamePromt(text, inifile) == true {
			return "That prompt is already registered", "", -1
		}
		return string(text), string(strd), 1
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
			return string(stra), "", 2
		}
	}

	sFlag := 0
	var stra string

	if strings.Index(text, like+" ") == 0 || strings.Index(text, like+"　") == 0 {
		sFlag = 10
		stra = strings.Replace(text, like, "", -1)
	} else if strings.Index(text, like3+" ") == 0 || strings.Index(text, like3+"　") == 0 {
		sFlag = 11
		stra = strings.Replace(text, like3, "", -1)
	} else if strings.Index(text, match+" ") == 0 || strings.Index(text, match+"　") == 0 {
		sFlag = 12
		stra = strings.Replace(text, match, "", -1)
	} else if strings.Index(text, match3+" ") == 0 || strings.Index(text, match3+"　") == 0 {
		sFlag = 13
		stra = strings.Replace(text, match3, "", -1)
	}

	if sFlag > 0 {
		_, count := readText(inifile, false, false)
		if count == 0 {
			return "no index exits!", "", -1
		}
		if len(stra) < 1 {
			return "Please specify search words", "", -1
		}
		return stra, "", sFlag
	}

	return "", "", -1
}

func checkSamePromt(prompt, inifile string) bool {
	str := rejectEscape(prompt)
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
	}

	if err = scanner.Err(); err != nil {
		fmt.Printf("scanner.Err: %#v\n", err)
		os.Exit(-1)
	}

	return false
}

func reLoad(filename string, count, Gramsize int) *ngram.NGramIndex {
	index, _ := ngram.NewNGramIndex(ngram.SetN(Gramsize))

	fp, err := os.Open(filename)
	if err != nil {
		fmt.Printf("os.Open: %#v\n", err)
		os.Exit(-1)
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)

	for scanner.Scan() {
		str := scanner.Text()
		index.Add(str)
		//debugLog("add Index: " + str)
	}

	if err = scanner.Err(); err != nil {
		fmt.Printf("scanner.Err: %#v\n", err)
		os.Exit(-1)
	}
	return index
}

func checkReaction(api *slack.Client, reactions []slack.ItemReaction, promptID string) {
	for _, reaction := range reactions {
		for _, user := range reaction.Users {
			debugLog(user + " " + getUsername(api, user) + " " + reaction.Name)
			actions = append(actions, promptAction{ID: promptID, USER: getUsername(api, user), ACTION: reaction.Name})
		}
	}
}

func getUsername(api *slack.Client, userID string) string {
	user, err := api.GetUserInfo(userID)
	if err != nil {
		fmt.Printf("%s\n", err)
		return ""
	}
	return user.Profile.RealName
}

func readText(filename string, sFlag, lFlag bool) (string, int) {
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
		strb := scanner.Text()
		if len(strb) > 1 {
			if lFlag == true {
				debugLog("timeLock: " + strb)
				str = str + strb
				break
			}
			if sFlag == true {
				str = str + strb + "\n"
			}
			line++
		}
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
