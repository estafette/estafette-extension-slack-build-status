package main

import (
	"fmt"
	stdlog "log"
	"os"
	"runtime"
	"strings"

	"github.com/alecthomas/kingpin"

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
	slackWebhookURL       = kingpin.Flag("slack-webhook-url", "A slack webhook url to allow sending messages.").Envar("ESTAFETTE_SLACK_WEBHOOK").Required().String()
	slackChannels         = kingpin.Flag("slack-channels", "A comma-separated list of Slack channels to send build status to.").Envar("ESTAFETTE_EXTENSION_CHANNELS").String()
	slackUsers            = kingpin.Flag("slack-users", "A comma-separated list of Slack users to send build status to.").Envar("ESTAFETTE_EXTENSION_USERS").String()
	gitName               = kingpin.Flag("git-name", "The owner plus repository name.").Envar("ESTAFETTE_GIT_NAME").Required().String()
	gitBranch             = kingpin.Flag("git-branch", "The branch to clone.").Envar("ESTAFETTE_GIT_BRANCH").Required().String()
	gitRevision           = kingpin.Flag("git-revision", "The revision to check out.").Envar("ESTAFETTE_GIT_REVISION").Required().String()
	estafetteBuildVersion = kingpin.Flag("estafette-build-version", "The current build version of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_VERSION").Required().String()
	estafetteBuildStatus  = kingpin.Flag("estafette-build-status", "The current build status of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_STATUS").Required().String()
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

	slackWebhookClient := NewSlackWebhookClient(*slackWebhookURL)

	if *slackChannels != "" {

		// set message depending on status
		message := ""
		switch *estafetteBuildStatus {
		case "succeeded":
			message = fmt.Sprintf("Build %v - repository %v for branch %v and revision %v - succeeded", *estafetteBuildVersion, *gitName, *gitBranch, *gitRevision)
		case "failed":
			message = fmt.Sprintf("Build %v - repository %v for branch %v and revision %v - failed", *estafetteBuildVersion, *gitName, *gitBranch, *gitRevision)
		}

		// split on comma and loop through channels
		channels := strings.Split(*slackChannels, ",")

		for i := range channels {
			err := slackWebhookClient.SendMessage(channels[i], message)
			if err != nil {
				log.Error().Err(err).Msg("Sending build status to Slack failed")
				os.Exit(1)
			}
		}
	}

	log.Info().Msg("Finished estafette-extension-slack-build-status...")
}
