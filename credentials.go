package main

import "log"

// SlackCredentials are credentials defined in the CI server and injected into this trusted image
type SlackCredentials struct {
	Name                 string                               `json:"name,omitempty"`
	Type                 string                               `json:"type,omitempty"`
	AdditionalProperties SlackCredentialsAdditionalProperties `json:"additionalProperties,omitempty"`
}

// SlackCredentialsAdditionalProperties has additional properties for the slack credentials
type SlackCredentialsAdditionalProperties struct {
	Workspace string `json:"workspace,omitempty"`
	Webhook   string `json:"webhook,omitempty"`
}

// GetCredentialsByWorkspace returns a credential if one for the workspace exists
func GetCredentialsByWorkspace(c []SlackCredentials, workspace string) *SlackCredentials {

	for _, cred := range c {
		log.Printf("Checking credential %v for workspace %v...", cred.Name, workspace)
		if cred.AdditionalProperties.Workspace == workspace {
			log.Printf("Found credential %v for workspace %v", cred.Name, workspace)
			return &cred
		}
	}

	return nil
}
