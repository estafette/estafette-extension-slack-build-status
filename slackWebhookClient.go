package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/sethgrid/pester"
)

// SlackWebhookClient is used to send messages to slack using a webhook
type SlackWebhookClient interface {
	SendMessage(string, string, string, string, string, bool) error
}

type slackWebhookClientImpl struct {
	webhookURL string
}

// NewSlackWebhookClient returns a new SlackWebhookClient
func NewSlackWebhookClient(webhookURL string) SlackWebhookClient {
	return &slackWebhookClientImpl{
		webhookURL: webhookURL,
	}
}

// GetAccessToken returns an access token to access the Bitbucket api
func (sc *slackWebhookClientImpl) SendMessage(target, title, message, color, link string, showLogsButton bool) (err error) {

	var requestBody io.Reader

	slackMessageBody := SlackMessageBody{
		Channel:  target,
		Username: "Estafette CI",
		Attachments: []SlackMessageAttachment{
			SlackMessageAttachment{
				Fallback:  message,
				Title:     title,
				TitleLink: link,
				Text:      message,
				Color:     color,
				MarkdownIn: []string{
					"text",
				},
			},
		},
	}

	if showLogsButton {
		slackMessageBody.Attachments[0].Actions = []SlackMessageAction{
			SlackMessageAction{
				Type:  "button",
				Text:  "View logs",
				URL:   link,
				Style: "primary",
			},
		}
	}

	data, err := json.Marshal(slackMessageBody)
	if err != nil {
		log.Printf("Failed marshalling SlackMessageBody: %v. Error: %v", slackMessageBody, err)
		return
	}
	requestBody = bytes.NewReader(data)

	// create client, in order to add headers
	client := pester.New()
	client.MaxRetries = 3
	client.Backoff = pester.ExponentialJitterBackoff
	client.KeepLog = true
	request, err := http.NewRequest("POST", sc.webhookURL, requestBody)
	if err != nil {
		log.Printf("Failed creating http client: %v", err)
		return
	}

	// add headers
	request.Header.Add("Content-type", "application/json")

	// perform actual request
	response, err := client.Do(request)
	if err != nil {
		log.Printf("Failed performing http request to Slack: %v", err)
		return
	}

	defer response.Body.Close()

	return
}
