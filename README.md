# prompt-bot
 **slack bot to manage prompts with organizational efficiency**

# Solution

I'd like to use ChatGPT internally to promote prompt engineering. But I don't know how to **manage company-wide** prompts...<br>
With the current mix of remote workers and those who come to work, the **Chat tool should be utilized as a hub of communication**.<br>
And because it's a Slack bot, it can be used from a smartphone!<br>
I tried to code such a thought!<br>

# Feature

The following operations can be performed on Slack by specifying a word!

- You can register a prompt in Slack and a sample of its output. (Samples are available for both **images and text**.)<br>
- You can search for registered prompts in natural language. (The search algorithm is **ngram**.)<br>

# installation

If you want to put it under the path, you can use the following.

```
go get github.com/yasutakatou/prompt-bot
```

If you want to create a binary and copy it yourself, use the following.

```
git clone https://github.com/yasutakatou/prompt-bot
cd prompt-bot
go build .
```

or download binary from release page. save binary file, copy to entryed execute path directory.

# uninstall

delete that binary. del or rm command. (it's simple!)

# set up

Please follow the steps below to set up your environment.

1. set tool like bot. 
- goto [slack api](https://api.slack.com/apps)
- Create New(an) App
	- define (Name)
	- select (Workspce)
	- Create App
- App-Level Tokens
	- Generate Token and Scopes
	- define (Name)
	- Add Scope
		- connections:write
	- Generate
		- Make a note of the token that begins with xapp-.
	- Done
- Socket Mode
	- Enable Socket Mode
		- On
- OAuth & Permissions
	- Scopes
	- Bot Token Scopes
		- channels:history
		- chat:write
		- files:write
		- reactions:write
		- users:read
	- Install to Workspace
	- Bot User OAuth Token
		- Make a note of the token that begins with xoxb-.
- Event Subscriptions
	- Enable Events
		- On
	- Subscribe to bot events
	- Add Bot User Event
		- message.channels
	- Save Changes

2. on Slack App
	- invite bot
		- @(Name)
	- invite

## If you want to use it on a private channel

If you want to use **Private channnel**, add the following settings

- OAuth & Permissions
	- Scopes
	- Bot Token Scopes
		- **groups:history**
	- Install to Workspace
- Event Subscriptions
	- Subscribe to bot events
	- Add Bot User Event
		- **message.groups**
	- Save Changes

3. your OS terminal
	- set environment
		- windows
			- set SLACK_APP_TOKEN=xapp-...
			- set SLACK_BOT_TOKEN=xoxb-...
		- linux
			- export SLACK_APP_TOKEN=xapp-...
			- export SLACK_BOT_TOKEN=xoxb-...
	- run this tool

# usecase

There are three major uses<br>
1) Register prompts and output results in **text**.<br>
2) Register prompts as **text and output results as images**.<br>
3) **Search** prompts with natural language processing.<br>
<br>
note) Deletion is not supported. Because we haven't come up with an easy-to-understand deletion method.<br>
<br>
In other words, prompt sharing can be achieved on a company-wide basis by applying this bot across the board!<br>
<br>
note) It would be interesting to visualize the use of AI by aggregating the **most used prompts or most searched people from the usage history logs**.　You should honor its users.<br>

## When registering by text

The prompt and output results are arranged in text, bounded by the words specified in the **options record and result**.

![1](https://github.com/yasutakatou/prompt-bot/assets/22161385/712c0bca-47b9-4dee-8d9d-011ad427198e)

The bot will suggest the top-ranking prompts from ngram search results.

![image](https://github.com/yasutakatou/prompt-bot/assets/22161385/b65932cc-ff55-4d91-84ce-c998e80bedda)

## To register the results as an image

**Attached is an image** of the resulting output along with a word for prompt registration.

![1](https://github.com/yasutakatou/prompt-bot/assets/22161385/b967e745-7aff-4653-a120-22c18810d5da)

Here, too, the bot suggests top prompts from ambiguous search results.

![2](https://github.com/yasutakatou/prompt-bot/assets/22161385/e3fc165a-54a4-49cd-aa83-1c1b0fbc09f8)

## Output usage reports

There is the ability to periodically go back through the messages and **tally the reactions to the prompts**.

![3](https://github.com/yasutakatou/prompt-bot/assets/22161385/5731112e-3e08-44b5-919f-2e34236df7d8)

This would also stimulate organizational use by recognizing those who have posted the best prompts based on the number of reactions they have received, or those who have added the most reactions (i.e., those who are making the most use of the prompts)!

![4](https://github.com/yasutakatou/prompt-bot/assets/22161385/b3fd005d-1318-4796-ab31-676f6bfed45c)

# options

```
  -Gramsize int
        [-Gramsize=N (gram size) to NGramIndex c-tor.] (default 3)
  -botid string
        [-botid=Define IDs for bots to prevent response loops. (default "U026G2JFYC9")
  -debug
        [-debug=debug mode (true is enable)]
  -dir string
        [-dir=Directory to store registered information. (default "data")
  -historySize int
        [-historySize=Specify the number of statements to look back on channel usage.] (default 100)
  -ini string
        [-ini=config file name. (default "prompt-bot.ini")
  -like string
        [-like=The word when searching for prompts.] (default "like")
  -like3 string
        [-like3=The word when searching for prompts.] (default "like3")
  -log
        [-log=logging mode (true is enable)]
  -loop int
        [-loop=The interval at which periodic usage reports are output.] (default 24)
  -match string
        [-match=The word when searching for prompts.] (default "match")
  -match3 string
        [-match3=The word when searching for prompts.] (default "match3")
  -noreport
        [-debug=Put it in a mode that does not output periodic usage reports.]
  -record string
        [-record=These are the words used to register the prompt] (default "record")
  -reportChannel string
        [-reportChannel=Specify which channels to output periodic usage reports.] (default "XXXXXXXX")
  -result string
        [-result=A word that specifies the output of the prompt] (default "result")
  -threshold string
        [-threshold=Threshold for best matching sentences.] (default "0.2")
  -top int
        [-top=Change to a number other than TOP 3.] (default 3)
```

## -Gramsize

N (gram size) to NGramIndex c-tor. (default 3)<br>
<br>
note) If you make it too big, it will crash due to memory error. 3 is good.<br>

## -botid

Option to define the bot's own ID<br>
<br>
note) Define bots not to respond to their own posts.<br>
<br>
![image](https://github.com/yasutakatou/prompt-bot/assets/22161385/b6a9972f-420e-40c8-a885-0d7bb5f00ed8)

```
https://app.slack.com/team/U026G2JFYC9
```

**U026G2JFYC9** is bot ID.

## -debug

Run in the mode that outputs various logs.

## -dir

Output directory for various data.

## -historySize

Specify the number of statements to look back on channel usage.<br>
<br>
note) If the daily search volume amounts to several hundred searches, please increase it.<br>

## -ini

Specify the configuration file name.

## -like

Prompt to invoke an ambiguous search(n-gram).

##  -like3

Prompt for returning multiple fuzzy searchh(n-gram) results.

## -log

Specify the log file name.

## -loop

The interval at which periodic usage reports are output. (default 24 Hour)

## -match

Prompt for static word search.

## -match3

Prompt for multiple static word searches.

## -noreport

Put it in a mode that does not output periodic usage reports.

## -record

These are the words used to register the prompt (default "record")

## -reportChannel

Specify which channels to output periodic usage reports.

![image](https://github.com/yasutakatou/prompt-bot/assets/22161385/57b4d590-046b-4905-812a-f490a9fa503d)

## -result string

A word that specifies the output of the prompt (default "result")

## -threshold string

Criterion value for determining similarity<br>
<br>
note) The lower the similarity, the less likely it is to be a candidate.<br>

## -top

Change to a number other than TOP 3.<br>
<br>
If you change this, it is recommended that you also change the prompt name for multiple searches.<br>

# License
MIT License


##
