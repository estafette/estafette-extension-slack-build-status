package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/alecthomas/kingpin"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/rs/zerolog/log"
)

var (
	appgroup  string
	app       string
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
//   - 'curl -X POST --data-urlencode ''payload={"channel": "#build-status", "username": "estafette-extension-slack-build-status", "text": "Build ''${ESTAFETTE_BUILD_VERSION}'' for ''${ESTAFETTE_GIT_NAME}'' has failed!"}'' ${ESTAFETTE_SLACK_WEBHOOK}'
//   when:
//     status == 'failed'

// TO:
//
// slack-notify:
//   image: extensions/slack-build-status:dev
//   workspace: estafette
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
	slackTitle               = kingpin.Flag("slack-title", "A custom slack message title.").Envar("ESTAFETTE_EXTENSION_TITLE").String()
	buildName                = kingpin.Flag("build-name", "The name of the pipeline that succeeds or fails.").Envar("ESTAFETTE_EXTENSION_NAME").String()

	gitName  = kingpin.Flag("git-name", "Repository name, used as application name if not passed explicitly and app label not being set.").Envar("ESTAFETTE_GIT_NAME").String()
	appLabel = kingpin.Flag("app-name", "App label, used as application name if not passed explicitly.").Envar("ESTAFETTE_LABEL_APP").String()

	ciBaseURL          = kingpin.Flag("estafette-ci-server-base-url", "The base url of the ci server.").Envar("ESTAFETTE_CI_SERVER_BASE_URL").String()
	gitRepoSource      = kingpin.Flag("git-repo-source", "The source of the git repository, github.com in this case.").Envar("ESTAFETTE_GIT_SOURCE").Required().String()
	gitRepoFullname    = kingpin.Flag("git-repo-fullname", "The owner and repo name of the Github repository.").Envar("ESTAFETTE_GIT_FULLNAME").Required().String()
	estafetteBuildID   = kingpin.Flag("estafette-build-id", "The build id of this particular build.").Envar("ESTAFETTE_BUILD_ID").String()
	estafetteReleaseID = kingpin.Flag("estafette-release-id", "The release id of this particular release.").Envar("ESTAFETTE_RELEASE_ID").String()

	estafetteBuildVersion = kingpin.Flag("estafette-build-version", "The current build version of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_VERSION").Required().String()
	estafetteBuildStatus  = kingpin.Flag("estafette-build-status", "The current build status of the Estafette pipeline.").Envar("ESTAFETTE_BUILD_STATUS").Required().String()
	statusOverride        = kingpin.Flag("status-override", "Allow status property in manifest to override the actual build status.").Envar("ESTAFETTE_EXTENSION_STATUS").String()

	workspace       = kingpin.Flag("slack-extension-workspace", "A slack workspace.").Envar("ESTAFETTE_EXTENSION_WORKSPACE").String()
	credentialsPath = kingpin.Flag("credentials-path", "Path to file with Slack credentials configured at server level, passed in to this trusted extension.").Default("/credentials/slack_webhook.json").String()

	releaseName   = kingpin.Flag("release-name", "Name of the release section, automatically set by Estafette CI.").Envar("ESTAFETTE_RELEASE_NAME").String()
	releaseAction = kingpin.Flag("release-action", "Name of the release action, automatically set by Estafette CI.").Envar("ESTAFETTE_RELEASE_ACTION").String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// init log format from envvar ESTAFETTE_LOG_FORMAT
	foundation.InitLoggingFromEnv(appgroup, app, version, branch, revision, buildDate)

	// make sure ciBaseURL ends with a slash
	if !strings.HasSuffix(*ciBaseURL, "/") {
		*ciBaseURL = *ciBaseURL + "/"
	}

	var credential *SlackCredentials
	if *slackWebhookURL == "" && *slackExtensionWebhookURL == "" {

		if *workspace != "" {
			log.Printf("Unmarshalling credentials...")
			var credentials []SlackCredentials

			// use mounted credential file if present instead of relying on an envvar
			if runtime.GOOS == "windows" {
				*credentialsPath = "C:" + *credentialsPath
			}
			if foundation.FileExists(*credentialsPath) {
				log.Info().Msgf("Reading credentials from file at path %v...", *credentialsPath)
				credentialsFileContent, err := ioutil.ReadFile(*credentialsPath)
				if err != nil {
					log.Fatal().Msgf("Failed reading credential file at path %v.", *credentialsPath)
				}
				err = json.Unmarshal(credentialsFileContent, &credentials)
				if err != nil {
					log.Fatal().Err(err).Msg("Failed unmarshalling injected credentials")
				}
			} else {
				log.Fatal().Msg("Credentials of type kubernetes-engine are not injected; configure this extension as trusted and inject credentials of type kubernetes-engine")
			}

			log.Printf("Checking if credential %v exists...", *workspace)
			credential = GetCredentialsByWorkspace(credentials, *workspace)
			if credential == nil {
				log.Fatal().Msgf("Credential with workspace %v does not exist.", *workspace)
			}
		} else {
			log.Fatal().Msg("Either flag slack-webhook-url or slack-extension-webhook has to be set")
		}
	}

	// set defaults
	if *buildName == "" && *appLabel == "" && *gitName != "" {
		*buildName = *gitName
	}
	if *buildName == "" && *appLabel != "" {
		*buildName = *appLabel
	}

	// pick via whatever method the webhook url has been set
	webhookURL := *slackWebhookURL
	if credential != nil {
		// log.Printf("Setting webhook from credential: %v", credential.AdditionalProperties.Webhook)
		webhookURL = credential.AdditionalProperties.Webhook
	} else if *slackExtensionWebhookURL != "" {
		log.Print("Overriding slackWebhookURL with slackExtensionWebhookURL")
		webhookURL = *slackExtensionWebhookURL
	}

	slackWebhookClient := NewSlackWebhookClient(webhookURL)

	if *slackChannels != "" {

		server := os.Getenv("ESTAFETTE_CI_SERVER")

		var logsURL string
		if server != "gocd" {
			logsURL = fmt.Sprintf(
				"%vpipelines/%v/%v/builds/%v/logs",
				*ciBaseURL,
				*gitRepoSource,
				*gitRepoFullname,
				*estafetteBuildID,
			)

			if *releaseName != "" {
				logsURL = fmt.Sprintf(
					"%vpipelines/%v/%v/releases/%v/logs",
					*ciBaseURL,
					*gitRepoSource,
					*gitRepoFullname,
					*estafetteReleaseID,
				)
			}
		}

		// check if there's a status override
		status := *estafetteBuildStatus
		if *statusOverride != "" {
			status = *statusOverride
		}

		// set message depending on status
		title := fmt.Sprintf("Building %v %v!", *buildName, status)
		message := fmt.Sprintf("Build version %v %v.", *estafetteBuildVersion, status)
		if *releaseName != "" {
			title = fmt.Sprintf("Releasing %v to %v %v!", *buildName, *releaseName, status)
			message = fmt.Sprintf("Release %v to %v %v.", *estafetteBuildVersion, *releaseName, status)
			if *releaseAction != "" {
				title = fmt.Sprintf("Releasing %v to %v with %v %v!", *buildName, *releaseName, *releaseAction, status)
				message = fmt.Sprintf("Release %v to %v with %v %v.", *estafetteBuildVersion, *releaseName, *releaseAction, status)
			}
		}

		// apply custom title if provided
		if *slackTitle != "" {
			title = *slackTitle
		}

		// split on comma and loop through channels
		channels := strings.Split(*slackChannels, ",")

		for i := range channels {
			err := slackWebhookClient.SendMessage(channels[i], title, message, status, logsURL, server != "gocd")
			if err != nil {
				log.Printf("Sending status to Slack failed: %v", err)
				os.Exit(1)
			}
		}
	}

	log.Print("Finished estafette-extension-slack-build-status...")
}
