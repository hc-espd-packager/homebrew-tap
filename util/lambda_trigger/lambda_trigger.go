package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// ReleaseEvent event body
type ReleaseEvent struct {
	Product string `json:"product"`
	Version string `json:"version"`
	SHASUM  string `json:"shasum"`
}

func shouldTriggerWorkflow(product string) bool {
	supportedProducts := []string{"vault", "consul", "nomad", "terraform", "packer"}

	for _, p := range supportedProducts {
		if p == product {
			return true
		}
	}

	return false
}

func triggerGithubWorkflow(event *ReleaseEvent) error {
	githubToken := os.Getenv("GITHUB_TOKEN")
	// Create dispatch event https://docs.github.com/en/rest/reference/repos#create-a-repository-dispatch-event
	workflowEndpoint := "https://api.github.com/repos/hashicorp/homebrew-tap/dispatches"
	postBody := fmt.Sprintf("{\"event_type\": \"version-updated\", \"client_payload\":{\"name\":\"%s\",\"version\":\"%s\",\"shasum\":\"%s\"}}", event.Product, event.Version, event.SHASUM)
	fmt.Printf("POSTing to Github: %s\n", postBody)

	httpClient := &http.Client{}
	req, err := http.NewRequest("POST", workflowEndpoint, bytes.NewBufferString(postBody))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", githubToken))
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Printf("Github Response: %+v", body)
	return err
}

// HandleLambdaEvent handler function to trigger github workflow
func HandleLambdaEvent(snsEvent events.SNSEvent) error {
	for _, record := range snsEvent.Records {
		snsRecord := record.SNS
		fmt.Printf("[%s %s] Message = %s \n", record.EventSource, snsRecord.Timestamp, snsRecord.Message)
		message := record.SNS.Message

		// Parse message to ReleaseEvent type
		event := &ReleaseEvent{}
		err := json.Unmarshal([]byte(message), event)
		if err != nil {
			return err
		}

		if shouldTriggerWorkflow(event.Product) {
			err := triggerGithubWorkflow(event)

			return err
		}
	}

	return nil
}

func main() {
	lambda.Start(HandleLambdaEvent)
}
