package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/alecthomas/kingpin"
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
	buildName                = kingpin.Flag("build-name", "The name of the pipeline that succeeds or fails.").Envar("ESTAFETTE_EXTENSION_NAME").String()

	ciBaseURL          = kingpin.Flag("estafette-ci-server-base-url", "The base url of the ci server.").Envar("ESTAFETTE_CI_SERVER_BASE_URL").Required().String()
	gitRepoSource      = kingpin.Flag("git-repo-source", "The source of the git repository, github.com in this case.").Envar("ESTAFETTE_GIT_SOURCE").Required().String()
	gitRepoFullname    = kingpin.Flag("git-repo-fullname", "The owner and repo name of the Github repository.").Envar("ESTAFETTE_GIT_FULLNAME").Required().String()
	estafetteBuildID   = kingpin.Flag("estafette-build-id", "The build id of this particular build.").Envar("ESTAFETTE_BUILD_ID").String()
	estafetteReleaseID = kingpin.Flag("estafette-release-id", "The release id of this particular release.").Envar("ESTAFETTE_RELEASE_ID").String()

	estafetteBuildVersion = kingpin.Flag("estafette-build-version", "The current build version of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_VERSION").Required().String()
	estafetteBuildStatus  = kingpin.Flag("estafette-build-status", "The current build status of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_STATUS").Required().String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// log to stdout and hide timestamp
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// log startup message
	log.Printf("Starting estafette-extension-slack-build-status version %v...", version)

	if *slackWebhookURL == "" && *slackExtensionWebhookURL == "" {
		log.Fatal("Either flag slack-webhook-url or slack-extension-webhook has to be set")
	}

	// set defaults
	appLabel := os.Getenv("ESTAFETTE_LABEL_APP")
	if *buildName == "" && appLabel != "" {
		*buildName = appLabel
	}

	// pick via whatever method the webhook url has been set
	webhookURL := *slackWebhookURL
	if *slackExtensionWebhookURL != "" {
		log.Print("Overriding slackWebhookURL with slackExtensionWebhookURL")
		webhookURL = *slackExtensionWebhookURL
	}

	slackWebhookClient := NewSlackWebhookClient(webhookURL)

	if *slackChannels != "" {

		server := os.Getenv("ESTAFETTE_CI_SERVER")
		releaseName := os.Getenv("ESTAFETTE_RELEASE_NAME")

		var logsURL string
		if server != "gocd" {
			logsURL = fmt.Sprintf(
				"%vpipelines/%v/%v/builds/%v/logs",
				*ciBaseURL,
				*gitRepoSource,
				*gitRepoFullname,
				*estafetteBuildID,
			)

			if releaseName != "" {
				logsURL = fmt.Sprintf(
					"%vpipelines/%v/%v/releases/%v/logs",
					*ciBaseURL,
					*gitRepoSource,
					*gitRepoFullname,
					*estafetteReleaseID,
				)
			}
		}

		// set message depending on status
		title := fmt.Sprintf("Building %v %v!", *buildName, *estafetteBuildStatus)
		message := fmt.Sprintf("Building version *%v* of *%v* %v.", *estafetteBuildVersion, *buildName, *estafetteBuildStatus)
		if releaseName != "" {
			title = fmt.Sprintf("Releasing %v to %v %v!", *buildName, releaseName, *estafetteBuildStatus)
			message = fmt.Sprintf("Releasing *%v* of *%v* to *%v* %v.", *estafetteBuildVersion, *buildName, releaseName, *estafetteBuildStatus)
		}

		if server != "gocd" {
			message += fmt.Sprintf(" <%v|See logs for more information>.", logsURL)
		}

		color := ""
		switch *estafetteBuildStatus {
		case "succeeded":
			color = "good"
		case "failed":
			color = "danger"
		}

		// split on comma and loop through channels
		channels := strings.Split(*slackChannels, ",")

		for i := range channels {
			err := slackWebhookClient.SendMessage(channels[i], title, message, color, logsURL)
			if err != nil {
				log.Printf("Sending status to Slack failed: %v", err)
				os.Exit(1)
			}
		}
	}

	log.Print("Finished estafette-extension-slack-build-status...")
}
