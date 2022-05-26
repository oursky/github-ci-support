package main

import (
	"context"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env.local", ".env")

	stop := make(chan struct{})
	defer close(stop)

	store, err := NewStore(context.Background(), os.Getenv("APP_KUBE_NAMESPACE"), os.Getenv("APP_KUBE_CONFIG_NAME"))
	if err != nil {
		panic(err)
	}
	store.Start(stop)

	slack := NewSlack(os.Getenv("APP_SLACK_BOT_TOKEN"), os.Getenv("APP_SLACK_APP_TOKEN"))

	appID, err := strconv.ParseInt(os.Getenv("APP_GITHUB_APP_ID"), 10, 64)
	if err != nil {
		panic(err)
	}
	webhook, err := NewGitHubWebhook(slack, store, appID, os.Getenv("APP_GITHUB_WEBHOOK_SECRET"))
	if err != nil {
		panic(err)
	}

	webhook.Start()
	err = slack.RunSocket(store)
	if err != nil {
		panic(err)
	}
}
