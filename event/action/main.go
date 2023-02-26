package main

import (
	"context"
	"log"
	"os"
	"regexp"

	l "github.com/aws/aws-lambda-go/lambda"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func Handler(ctx context.Context, input *slackevents.AppMentionEvent) {
	cli := slack.New(os.Getenv("SLACK_ACCESS_TOKEN"))
	_, _, err := cli.PostMessage(
		input.Channel,
		slack.MsgOptionText(
			"RE: "+textWithMentionsRemoved(input.Text), false,
		),
	)
	if err != nil {
		log.Printf("failed to post message: %s", err)
	}
}

func textWithMentionsRemoved(txt string) string {
	re := regexp.MustCompile("<@.+?>")
	return re.ReplaceAllString(txt, "")
}

func main() {
	l.Start(Handler)
}
