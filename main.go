package main

import (
	"bytes"
	"encoding/json"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"runtime"

	"github.com/alecthomas/kingpin"
	"github.com/sethgrid/pester"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

// FROM:
//
// slack-notify:
//   image: golang:1.8.3-alpine3.6
//   commands:
//   - apk --update add curl
//   - 'curl -X POST --data-urlencode ''payload={"channel": "#build-status", "username": "estafette-extension-slack-build-status", "text": "Build ''${ESTAFETTE_BUILD_VERSION}'' for ''${ESTAFETTE_LABEL_APP}'' has failed!"}'' ${ESTAFETTE_SLACK_WEBHOOK}'
//   when:
//     status == 'failed'

// TO:
//
// slack-notify:
//   image: extensions/slack-build-status:dev
//   channels:
//   - "#build-status"
//   users:
//   - "@username"
//   when:
//     status == 'failed'

var (
	// flags
	slackWebhookURL      = kingpin.Flag("slack-webhook-url", "A slack webhook url to allow sending messages.").Envar("ESTAFETTE_SLACK_WEBHOOK").Required().String()
	estafetteBuildStatus = kingpin.Flag("estafette-build-status", "The current build status of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_STATUS").Required().String()
	slackChannels        = kingpin.Flag("slack-channels", "A comma-separated list of Slack channels to send build status to.").Envar("ESTAFETTE_EXTENSION_CHANNELS").String()
	slackUsers           = kingpin.Flag("slack-users", "A comma-separated list of Slack users to send build status to.").Envar("ESTAFETTE_EXTENSION_USERS").String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// pretty print to make build logs more readable
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().
		Timestamp().
		Logger()

	// use zerolog for any logs sent via standard log library
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	// log startup message
	log.Info().
		Str("branch", branch).
		Str("revision", revision).
		Str("buildDate", buildDate).
		Str("goVersion", goVersion).
		Msg("Starting estafette-extension-slack-build-status...")

	var requestBody io.Reader

	slackMessageBody := SlackMessageBody{}
	data, err := json.Marshal(slackMessageBody)
	if err != nil {
		log.Error().Err(err).Interface("body", slackMessageBody).Msg("Failed marshalling SlackMessageBody")
		return
	}
	requestBody = bytes.NewReader(data)

	// create client, in order to add headers
	client := pester.New()
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	request, err := http.NewRequest("POST", *slackWebhookURL, requestBody)
	if err != nil {
		log.Error().Err(err).Msg("Failed creating http client")
		return
	}

	// add headers
	//request.Header.Add("X-Estafette-Event", eventType)

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		log.Error().Err(err).Msg("Failed performing http request to Slack")
		return
	}

	defer response.Body.Close()

	log.Info().
		Msg("Finished estafette-extension-slack-build-status...")
}
