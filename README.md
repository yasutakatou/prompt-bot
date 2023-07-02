# prompt-bot
 **slack bot to manage prompts with organizational efficiency**

# Solution

I'd like to use ChatGPT internally to promote prompt engineering. But I don't know how to **manage company-wide** prompts...<br>
With the current mix of remote workers and those who come to work, the **Chat tool should be utilized as a hub of communication**.<br>
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
1) Register prompts and output results in text.<br>
2) Register prompts as text and output results as images.<br>
3) Search prompts with natural language processing.<br>
<br>
In other words, prompt sharing can be achieved on a company-wide basis by applying this bot across the board!<br>
<br>
note) It would be interesting to visualize the use of AI by aggregating the most used prompts or most searched people from the usage history logs<br>

## When registering by text

The prompt and output results are arranged in text, bounded by the words specified in the options record and result.

![1](https://github.com/yasutakatou/prompt-bot/assets/22161385/712c0bca-47b9-4dee-8d9d-011ad427198e)

The bot will suggest the top-ranking prompts from the fuzzy search results.

![image](https://github.com/yasutakatou/prompt-bot/assets/22161385/b65932cc-ff55-4d91-84ce-c998e80bedda)

## To register the results as an image

Attached is an image of the resulting output along with a word for prompt registration

![2](https://github.com/yasutakatou/prompt-bot/assets/22161385/a8e26a03-8d04-47f2-9240-bdf1540bb906)

Here, too, the bot suggests top prompts from ambiguous search results.

![3](https://github.com/yasutakatou/prompt-bot/assets/22161385/cc101e0b-8de8-48c4-8366-ab511d968250)


# options

```
  -botid string
        [-botid=Define IDs for bots to prevent response loops. (default "U026G2JFYC9")
  -debug
        [-debug=debug mode (true is enable)]
  -dir string
        [-dir=Directory to store registered information. (default "data")
  -ini string
        [-ini=config file name. (default "prompt-bot.ini")
  -log
        [-log=logging mode (true is enable)]
  -record string
        [-record=These are the words used to register the prompt] (default "record")
  -result string
        [-result=A word that specifies the output of the prompt] (default "result")
  -search string
        [-search=The word when searching for prompts.] (default "search")
  -threshold string
        [-threshold=Threshold for best matching sentences.] (default "0.2")
```

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

## -ini string

Specify the configuration file name.

## -log

Specify the log file name.

## -record string

These are the words used to register the prompt (default "record")

## -result string

A word that specifies the output of the prompt (default "result")

## -search string

The word when searching for prompts. (default "search")

## -threshold string

Criterion value for determining similarity<br>
<br>
note) The lower the similarity, the less likely it is to be a candidate.<br>

# License
MIT License


##
