package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	l "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	eventsAPIEvent, err := verifyEvent(request)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return events.APIGatewayProxyResponse{
			Body:       request.Body,
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		response, err := returnChallengeValue(request)
		if err != nil {
			log.Printf("ERROR: %s", err)
			return events.APIGatewayProxyResponse{
				Body:       request.Body,
				StatusCode: http.StatusInternalServerError,
			}, nil
		}
		return response, nil
	case slackevents.AppRateLimited:
		log.Printf("Rate Limit ERROR: %s", request.Body)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
		}, nil
	default:
		// noop and proceed
	}

	data := eventsAPIEvent.InnerEvent.Data.(*slackevents.AppMentionEvent)
	input, err := buildInvokeInput(data)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return events.APIGatewayProxyResponse{
			Body:       request.Body,
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	sess, err := session.NewSession()
	if err != nil {
		log.Printf("ERROR: %s", err)
		return events.APIGatewayProxyResponse{
			Body:       request.Body,
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	ctx := aws.BackgroundContext()
	svc := lambda.New(sess)
	_, err = svc.InvokeWithContext(ctx, input)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return events.APIGatewayProxyResponse{
			Body:       request.Body,
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	response := events.APIGatewayProxyResponse{
		Body: "OK",
		Headers: map[string]string{
			"Content-Type": "text",
		},
		StatusCode: http.StatusOK,
	}

	return response, nil
}

func buildInvokeInput(data *slackevents.AppMentionEvent) (*lambda.InvokeInput, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	invokedFunction := os.Getenv("INVOKED_FUNCTION_NAME")
	input := &lambda.InvokeInput{
		FunctionName:   aws.String(invokedFunction),
		InvocationType: aws.String(lambda.InvocationTypeEvent),
		Payload:        b,
	}

	return input, nil
}

func verifyEvent(request events.APIGatewayProxyRequest) (slackevents.EventsAPIEvent, error) {
	if err := verifyRequest(request); err != nil {
		return slackevents.EventsAPIEvent{}, err
	}

	eventsAPIEvent, err := slackevents.ParseEvent(
		json.RawMessage(request.Body),
		slackevents.OptionVerifyToken(
			&slackevents.TokenComparator{
				VerificationToken: os.Getenv("SLACK_VERIFICATION_TOKEN"),
			},
		),
	)
	if err != nil {
		return slackevents.EventsAPIEvent{}, err
	}

	return eventsAPIEvent, nil
}

func verifyRequest(request events.APIGatewayProxyRequest) error {
	b := request.Body
	headers := convertHeaders(request.Headers)
	sv, err := slack.NewSecretsVerifier(headers, os.Getenv("SLACK_SIGNING_SECRET"))
	if err != nil {
		return err
	}

	_, err = sv.Write([]byte(b))
	if err != nil {
		return err
	}

	return sv.Ensure()
}

func convertHeaders(headers map[string]string) http.Header {
	h := http.Header{}
	for key, value := range headers {
		h.Set(key, value)
	}
	return h
}

func returnChallengeValue(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var r *slackevents.ChallengeResponse
	err := json.Unmarshal([]byte(request.Body), &r)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		Body: r.Challenge,
		Headers: map[string]string{
			"Content-Type": "text",
		},
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	l.Start(Handler)
}
