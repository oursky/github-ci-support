package main

import (
	"context"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/oursky/github-ci-support/githublib"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	syncInterval time.Duration = 10 * time.Second
	syncPageSize int           = 100
)

type RemoteRunner struct {
	ID       int64
	Name     string
	IsOnline bool
}

type RemoteRunners struct {
	BeginTime time.Time
	Epoch     int64
	Runners   map[string]RemoteRunner
}

func (r *RemoteRunners) Lookup(name string, id int64) (*RemoteRunner, bool) {
	if name == "" {
		return nil, false
	}
	if runner, ok := r.Runners[name]; ok {
		// id == 0 -> configuring, lookup by name
		if runner.ID == id || id == 0 {
			return &runner, true
		}
	}
	return nil, false
}

type Synchronizer struct {
	logger *zap.SugaredLogger
	target githublib.RunnerTarget
	client *github.Client
}

func NewSynchronizer(logger *zap.SugaredLogger, target githublib.RunnerTarget, client *github.Client) *Synchronizer {
	return &Synchronizer{
		logger: logger.Named("sync"),
		target: target,
		client: client,
	}
}

func (s *Synchronizer) Run(ctx context.Context, g *errgroup.Group, result chan<- *RemoteRunners) {
	g.Go(func() error {
		s.run(ctx, result)
		return nil
	})
}

func (s *Synchronizer) run(ctx context.Context, cResult chan<- *RemoteRunners) {
	epoch := int64(1)
	beginTime := time.Now()
	runners := make(map[string]RemoteRunner)
	page := 1

	for {
		select {
		case <-ctx.Done():
			close(cResult)
			return

		case <-time.After(syncInterval):
			s.logger.Infow("fetching page", "page", page)
			runnersPage, nextPage, err := s.target.GetRunners(ctx, s.client, page, syncPageSize)
			if err != nil {
				s.logger.Warnw("failed to get runners", "error", err)
				break
			}

			for _, r := range runnersPage {
				runners[r.GetName()] = RemoteRunner{
					ID:       r.GetID(),
					Name:     r.GetName(),
					IsOnline: r.GetStatus() == "online",
				}
			}

			if nextPage != 0 {
				page = nextPage
				break
			}

			s.logger.Infow("runners synchronized",
				"epoch", epoch,
				"beginTime", beginTime,
				"count", len(runners),
			)
			result := &RemoteRunners{BeginTime: beginTime, Epoch: epoch, Runners: runners}
			select {
			case cResult <- result:
			case <-ctx.Done():
				close(cResult)
				return
			}

			epoch++
			beginTime = time.Now()
			runners = make(map[string]RemoteRunner)
			page = 1
		}
	}
}
