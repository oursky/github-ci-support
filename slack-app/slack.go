package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

var repoRegex = regexp.MustCompile("[a-zA-Z0-9-]+(/[a-zA-Z0-9-]+)?")

type Slack struct {
	api *slack.Client
}

func NewSlack(botToken string, appToken string) *Slack {
	api := slack.New(
		botToken,
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(appToken),
	)
	return &Slack{api}
}

func (s *Slack) RunSocket(store *Store) error {
	client := socketmode.New(
		s.api,
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeHello:
				log.Println("Hello!")
			case socketmode.EventTypeConnecting:
				log.Println("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				log.Println("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				log.Println("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeSlashCommand:
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					log.Printf("Ignored %+v", evt)
					continue
				}

				log.Printf(
					"Channel: %s (%s); User: %s; Command: %s; Text: %s",
					cmd.ChannelName,
					cmd.ChannelID,
					cmd.UserName,
					cmd.Command,
					cmd.Text,
				)

				if cmd.Command != "/gha" {
					client.Ack(*evt.Request, map[string]interface{}{
						"text": fmt.Sprintf("Unknown command '%s'\n", cmd.Command)})
					return
				}

				subcommand, repo, _ := strings.Cut(cmd.Text, " ")
				if !repoRegex.MatchString(repo) {
					client.Ack(*evt.Request, map[string]interface{}{
						"text": fmt.Sprintf("Invalid repo '%s'\n", repo)})
					return
				}

				switch subcommand {
				case "subscribe":
					err := store.AddChannel(repo, cmd.ChannelID)
					if err != nil {
						log.Printf("failed to subscribe: %s", err)
						client.Ack(*evt.Request, map[string]interface{}{
							"text": fmt.Sprintf("Failed to subscribe '%s': %s\n", repo, err)})
					} else {
						client.Ack(*evt.Request, map[string]interface{}{
							"text": fmt.Sprintf("Subscribed to '%s'\n", repo)})
					}

				case "unsubscribe":
					err := store.DelChannel(repo, cmd.ChannelID)
					if err != nil {
						log.Printf("failed to unsubscribe: %s", err)
						client.Ack(*evt.Request, map[string]interface{}{
							"text": fmt.Sprintf("Failed to unsubscribe '%s': %s\n", repo, err)})
					} else {
						client.Ack(*evt.Request, map[string]interface{}{
							"text": fmt.Sprintf("Unsubscribed from '%s'\n", repo)})
					}

				default:
					client.Ack(*evt.Request, map[string]interface{}{
						"text": fmt.Sprintf("Unknown subcommand '%s'\n", subcommand)})
				}
			default:
				log.Printf("Unexpected event: %+v", evt.Type)
			}
		}
	}()

	return client.Run()
}

func (s *Slack) SendMessage(channelID string, opts ...slack.MsgOption) error {
	_, _, _, err := s.api.SendMessage(channelID, opts...)
	return err
}
