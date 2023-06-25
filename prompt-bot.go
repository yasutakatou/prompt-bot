package main

import (
	"bufio"
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

	index := ngram.NewIndex(getLines(*_Ini))
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
						str, eflag := validMessage(event.Text, *_Record, *_Result, *_Search)
						switch eflag {
						case 0:
							matches := index.FindBestMatch(str)
							fmt.Printf("matched %s\n", matches)
							strb := strings.Split(matches, "\t")

						case 1:
						case 2:
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

func validMessage(text, record, result, search string) (string, int) {
	if strings.Index(text, search) == 0 {
		stra := strings.Replace(text, search, "", -1)
		if len(stra) < 1 {
			return "", 10
		}
		return stra, 0
	}
	return "", 10
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

func getLines(filename string) int {
	line := 0

	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("os.Open: %#v\n", err)
		os.Exit(-1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line++
	}

	if err = scanner.Err(); err != nil {
		fmt.Printf("scanner.Err: %#v\n", err)
		os.Exit(-1)
	}

	fmt.Printf("lines: %d\n", line)
	return line
}
