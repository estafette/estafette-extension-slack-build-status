package main

// SlackMessageBody represents the body to send to slack
type SlackMessageBody struct {
	Channel  string `json:"channel"`
	Username string `json:"username"`
	Text     string `json:"text"`
}
