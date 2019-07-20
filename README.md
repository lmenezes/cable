# cable

<img width="300" alt="Screenshot 2019-06-25 at 17 45 14" src="https://user-images.githubusercontent.com/210307/61563551-8709b780-aa74-11e9-84f0-185e860a5bfe.png">

Slack <-> Telegram gateway

## Development

**cable** is developed in golang and ready to be deployed to Coogle Cloud Run.

**cable** is a service that listens to and relays messages between slack and telegram channels.

It's not intended to be a fully-fledged product, but instead solving a particular use case. 

## Installing dependencies

[Download and install Google Cloud SDK](https://cloud.google.com/sdk/)

Install and update components:

```sh
gcloud components install beta
gcloud components update
```

## Deploying

[Create a project](https://cloud.google.com/resource-manager/docs/creating-managing-projects) in google cloud engine.

open a shell and export the project id, under the `GCE_PROJECT_ID` environment vaiable.

<!-- TODO: make deploy run build if :latest image is not built -->

```sh
export GCE_PROJECT_ID=YOUR_PROJECT_ID make deploy
```

## Setting up slack and telegram integrations

```
gcloud beta run services update SERVICE_NAME --platform managed --region us-central1 --update-env-vars SLACK_TOKEN=´YOUR_SLACK_TOKEN,TELEGRAM_TOKEN=`YOUR_TELEGRAM_TOKEN` ... 
```

Other ENV variables that you might want configure until a UI is provided are:

* `SLACK_RELAYED_CHANNEL`  a string representing the ID of the Slack channel to relay messages to.
* `SLACK_BOT_ID` a string representing the ID of the cable slack application, to discard relaying their messages.
* `TELEGRAM_RELAYED_CHANNEL` an integer representing the ID of the Telegram conversation to relay messages to.
* `TELEGRAM_BOT_ID` an integer representing the ID of the cable telegram application, to discard relaying their messages.

## Supported features

* Bidireccional message relay: ✅
* Emoji: ✅
* Message edits: ❌
* Threads: ❌
* Reactions: ❌

## Licensed

This project is released under the [MIT LICENSE](LICENSE). Please note it includes 3rd party dependencies release under their own licenses; these are found under [vendor](https://github.com/github/freno/tree/master/vendor).
