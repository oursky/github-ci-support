package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v44/github"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackutilsx"
)

type GitHubWebhook struct {
	server *http.Server
	slack  *Slack
	appID  int64
	store  *Store
	secret []byte
}

func NewGitHubWebhook(slack *Slack, store *Store, appID int64, secret string) (*GitHubWebhook, error) {
	webhook := &GitHubWebhook{
		server: &http.Server{
			Addr:         ":8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		slack:  slack,
		store:  store,
		appID:  appID,
		secret: []byte(secret),
	}
	webhook.server.Handler = http.HandlerFunc(webhook.serveHTTP)

	return webhook, nil
}

func (s *GitHubWebhook) Start() {
	go func() {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("webhook: %v", err)
		}
	}()
}

func (s *GitHubWebhook) serveHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, s.secret)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
	}

	switch event := event.(type) {
	case *github.WorkflowRunEvent:
		s.handleEvent(event)
	}
}

func (s *GitHubWebhook) handleEvent(ev *github.WorkflowRunEvent) {
	if ev.GetAction() != "completed" {
		return
	}

	owner := ev.GetWorkflowRun().GetRepository().GetOwner().GetLogin()
	repoName := ev.GetWorkflowRun().GetRepository().GetName()
	repo := ev.GetWorkflowRun().GetRepository().GetFullName()

	workflow := ev.GetWorkflowRun().GetName()
	url := ev.GetWorkflowRun().GetHTMLURL()
	runID := ev.GetWorkflowRun().GetID()
	conclusion := ev.GetWorkflowRun().GetConclusion()

	commitMsg := ev.GetWorkflowRun().GetHeadCommit().GetMessage()
	commitURL := ev.GetWorkflowRun().GetHeadRepository().GetHTMLURL() + "/commit/" + ev.GetWorkflowRun().GetHeadCommit().GetID()

	log.Printf("Workflow run completed: %d @ %s", runID, repo)

	channels := s.store.GetChannels(repo)
	if len(channels) == 0 {
		log.Printf("Skipped.")
	}

	runTime, err := getRuntime(owner, repoName, runID,
		s.appID,
		ev.GetInstallation().GetID())
	if err != nil {
		log.Printf("failed to get timing: %s", err)
		runTime = "-"
	}

	var msg string = ""
	var color string = "#808080"
	switch conclusion {
	case "action_required":
		msg = fmt.Sprintf("%s requires action.", workflow)
		color = "#808000"
	case "cancelled":
		msg = fmt.Sprintf("%s is cancelled.", workflow)
	case "skipped":
		msg = ""
	case "failure":
		msg = fmt.Sprintf("%s has failed in %s.", workflow, runTime)
		color = "#800000"
	case "timed_out":
		msg = fmt.Sprintf("%s timed out in %s.", workflow, runTime)
		color = "#808000"
	case "success":
		msg = fmt.Sprintf("%s has succeeded in %s.", workflow, runTime)
		color = "#008000"
	default:
		msg = fmt.Sprintf("%s has completed in %s.", workflow, runTime)
	}

	if msg == "" {
		return
	}

	slackMsg := slack.Attachment{
		Color:      color,
		Title:      msg,
		TitleLink:  url,
		AuthorName: repo,
		MarkdownIn: []string{"fields"},
		Fields: []slack.AttachmentField{{
			Title: "Commit",
			Value: fmt.Sprintf(
				"<%s|%s>",
				slackutilsx.EscapeMessage(commitURL),
				slackutilsx.EscapeMessage(commitMsg),
			),
		}},
	}

	for _, channel := range channels {
		err := s.slack.SendMessage(channel, slack.MsgOptionAttachments(slackMsg))
		if err != nil {
			log.Printf("failed to send message to %s: %s", channel, err)
		}
	}
}

func getRuntime(owner string, repo string, runID int64, appID int64, installationID int64) (string, error) {
	itr, err := ghinstallation.NewKeyFromFile(
		http.DefaultTransport,
		appID,
		installationID,
		os.Getenv("APP_GITHUB_PRIVATE_KEY"))
	if err != nil {
		return "", err
	}

	client := github.NewClient(&http.Client{Transport: itr})
	timing, _, err := client.Actions.GetWorkflowRunUsageByID(context.Background(), owner, repo, runID)
	if err != nil {
		return "", err
	}

	return (time.Millisecond * time.Duration(timing.GetRunDurationMS())).String(), nil
}
