package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/oursky/github-ci-support/githublib"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")

	flag.Parse()

	if configPath == "" {
		panic("config is required")
	}

	l, _ := zap.NewProduction()
	defer l.Sync()
	logger := l.Sugar()

	config, err := NewConfig(configPath)
	if err != nil {
		panic(fmt.Sprintf("cannot load config: %s", err))
	}

	client, err := config.Auth.CreateClient()
	if err != nil {
		panic(fmt.Sprintf("cannot create client: %s", err))
	}

	target, err := githublib.NewRunnerTarget(config.Target)
	if err != nil {
		panic(fmt.Sprintf("cannot load target: %s", err))
	}

	token := githublib.NewRegistrationTokenStore(target, client)

	var runners []*Runner
	for i, runnerConfig := range config.Runners {
		runner := NewRunner(i, logger, config, runnerConfig, token)
		runners = append(runners, runner)
	}

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	go start(ctx, g, runners)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		logger.Info("exiting...")
		cancel()
	}()

	g.Wait()
}

func start(ctx context.Context, g *errgroup.Group, runners []*Runner) {
	for _, runner := range runners {
		runner.Run(ctx, g)
	}
}
