# cable

<img width="300" alt="Screenshot 2019-06-25 at 17 45 14" src="https://user-images.githubusercontent.com/210307/61563551-8709b780-aa74-11e9-84f0-185e860a5bfe.png">

Slack <-> Telegram gateway

## Development

**cable** is developed in golang and ready to be deployed to Heroku

**cable** is a service that listens to and relays messages between slack and telegram channels.

It's not intended to be a fully-fledged product, but instead solving a particular use case. 

## Setup cable

* [Create a telegram bot](https://core.telegram.org/bots#creating-a-new-bot) and add it to your conversations. Don't forget to 
[disable privacy mode](https://core.telegram.org/bots#privacy-mode) so the bot can listen to your conversations. 
* [Create a slack bot](https://api.slack.com/bot-users) and add it to your workspace
* Setup the appropriate environment variables:
	* `SLACK_TOKEN`  The api token to act on behalf of the slack bot. Slack will give you this information when you create the app
	* `SLACK_RELAYED_CHANNEL`  a string representing the ID of the Slack channel to relay messages to. [Get it from the `channels.list` api tester](https://api.slack.com/methods/channels.list/test)
	* `SLACK_BOT_ID` a string representing the ID of the cable slack application, to discard relaying their messages. [Get it from the `users.list` api tester](https://api.slack.com/methods/users.list/test)
	* `TELEGRAM_TOKEN`  The api token to act on behalf of the telegram bot. The BotFather will give you this information when you create the bot.
	* `TELEGRAM_RELAYED_CHANNEL` an integer representing the ID of the Telegram conversation to relay messages to.  [Learn how to get it, it's the `message.chat.id` field](https://stackoverflow.com/questions/32423837/telegram-bot-how-to-get-a-group-chat-id)
	* `TELEGRAM_BOT_ID` an integer representing the ID of the cable telegram application, to discard relaying their messages. [Learn how to get it, it's the `new_chat_participant.id` field](https://stackoverflow.com/questions/32423837/telegram-bot-how-to-get-a-group-chat-id)

## Deploy cable	

* Follow the tutorial on [deploying golang apps to heroku](https://devcenter.heroku.com/articles/getting-started-with-go)

## Supported features

* Bidireccional message relay: ✅
* Emoji: ✅
* Message edits: ❌
* Threads: ❌
* Reactions: ❌

## Licensed

This project is released under the [MIT LICENSE](LICENSE). Please note it includes 3rd party dependencies release under their own licenses; these are found under [vendor](https://github.com/github/freno/tree/master/vendor).
