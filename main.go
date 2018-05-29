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
	slackWebhookURL          = kingpin.Flag("slack-webhook-url", "A slack webhook url to allow sending messages.").Envar("ESTAFETTE_SLACK_WEBHOOK").String()
	slackExtensionWebhookURL = kingpin.Flag("slack-extension-webhook", "A slack webhook url to allow sending messages.").Envar("ESTAFETTE_EXTENSION_WEBHOOK").String()
	slackChannels            = kingpin.Flag("slack-channels", "A comma-separated list of Slack channels to send build status to.").Envar("ESTAFETTE_EXTENSION_CHANNELS").Required().String()
	buildName                = kingpin.Flag("build-name", "The name of the pipeline that succeeds or fails.").Envar("ESTAFETTE_EXTENSION_NAME").Required().String()
	gitBranch                = kingpin.Flag("git-branch", "The branch to clone.").Envar("ESTAFETTE_GIT_BRANCH").Required().String()
	gitRevision              = kingpin.Flag("git-revision", "The revision to check out.").Envar("ESTAFETTE_GIT_REVISION").Required().String()
	estafetteBuildVersion    = kingpin.Flag("estafette-build-version", "The current build version of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_VERSION").Required().String()
	estafetteBuildStatus     = kingpin.Flag("estafette-build-status", "The current build status of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_STATUS").Required().String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// pretty print to make build logs more readable
	log.Logger = zerolog.New(os.Stdout).With().
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

	if *slackWebhookURL == "" && *slackExtensionWebhookURL == "" {
		log.Fatal().Msg("Either flag slack-webhook-url or slack-extension-webhook has to be set")
	}

	// pick via whatever method the webhook url has been set
	webhookURL := *slackWebhookURL
	if *slackExtensionWebhookURL != "" {
		log.Debug().Msg("Overriding slackWebhookURL with slackExtensionWebhookURL")
		webhookURL = *slackExtensionWebhookURL
	}

	slackWebhookClient := NewSlackWebhookClient(webhookURL)

	if *slackChannels != "" {

		// set message depending on status
		title := ""
		message := ""
		color := ""
		switch *estafetteBuildStatus {
		case "succeeded":
			title = fmt.Sprintf("Build %v succeeded!", *buildName)
			message = fmt.Sprintf("Build *%v* of *%v* - branch _%v_ and revision _%v_ - succeeded", *estafetteBuildVersion, *buildName, *gitBranch, *gitRevision)
			color = "good"
		case "failed":
			title = fmt.Sprintf("Build %v failed!", *buildName)
			message = fmt.Sprintf("Build *%v* of *%v* - branch _%v_ and revision _%v_ - failed", *estafetteBuildVersion, *buildName, *gitBranch, *gitRevision)
			color = "danger"
		}

		// split on comma and loop through channels
		channels := strings.Split(*slackChannels, ",")

		for i := range channels {
			err := slackWebhookClient.SendMessage(channels[i], title, message, color)
			if err != nil {
				log.Error().Err(err).Msg("Sending build status to Slack failed")
				os.Exit(1)
			}
		}
	}

	log.Info().Msg("Finished estafette-extension-slack-build-status...")
}
