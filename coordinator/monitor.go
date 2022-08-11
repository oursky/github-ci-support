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
	transitionTimeoutEpochs int64 = 10
)

type RunnerState string

const (
	RunnerStatePending     RunnerState = "pending"
	RunnerStateConfiguring RunnerState = "configuring"
	RunnerStateStarting    RunnerState = "starting"
	RunnerStateReady       RunnerState = "ready"
	RunnerStateTerminating RunnerState = "terminating"
)

type localRunner struct {
	instanceID uint32
	instance   *RunnerInstance
	isDead     bool

	epoch              int64
	lastTransitionTime time.Time
	state              RunnerState

	runnerName string
	runnerID   int64
}

func (r *localRunner) update(epoch int64, state RunnerState) {
	if r.state == state {
		return
	}
	r.epoch = epoch
	r.lastTransitionTime = time.Now()
	r.state = state
}

type Monitor struct {
	logger *zap.SugaredLogger
	target githublib.RunnerTarget
	client *github.Client

	localRunners map[uint32]*localRunner
	remote       *RemoteRunners

	messages chan any
}

func NewMonitor(logger *zap.SugaredLogger, target githublib.RunnerTarget, client *github.Client) *Monitor {
	return &Monitor{
		logger:       logger.Named("monitor"),
		target:       target,
		client:       client,
		localRunners: make(map[uint32]*localRunner),
		remote:       &RemoteRunners{Epoch: 0, BeginTime: time.Now(), Runners: nil},
		messages:     make(chan any),
	}
}

func (m *Monitor) Run(ctx context.Context, g *errgroup.Group) {
	syncContext, stopSync := context.WithCancel(context.Background())
	synchronizer := NewSynchronizer(m.logger, m.target, m.client)
	sync := make(chan *RemoteRunners)

	synchronizer.Run(syncContext, g, sync)
	g.Go(func() error {
		m.run(ctx, sync, stopSync)
		return nil
	})
}

func (m *Monitor) Post(msg any) {
	m.messages <- msg
}

func (m *Monitor) run(ctx context.Context, sync <-chan *RemoteRunners, stopSync func()) {
	exit := false

	for !exit {
		select {
		case <-ctx.Done():
			exit = true

		case remote := <-sync:
			m.remote = remote
			m.checkRunners()

		case msg := <-m.messages:
			m.handleMessage(msg)
		}
	}

	m.cleanupRunners()

	for len(m.localRunners) > 0 {
		select {
		case remote := <-sync:
			m.remote = remote
			m.checkRunners()

		case msg := <-m.messages:
			m.handleMessage(msg)
		}
	}

	stopSync()
}

func (m *Monitor) terminate(runner *localRunner) {
	isOverdue := (m.remote.Epoch - runner.epoch) > transitionTimeoutEpochs
	done := true
	if !runner.isDead {
		runner.instance.Terminate(isOverdue)
		done = false
	}

	if r, ok := m.remote.Lookup(runner.runnerName, runner.runnerID); ok {
		m.logger.Infow("unregistering runner",
			"runnerID", r.ID,
			"runnerName", runner.runnerName,
		)

		if err := m.target.DeleteRunner(context.Background(), m.client, r.ID); err != nil {
			m.logger.Warnw("failed to delete runner", "error", err)
			if isOverdue {
				m.logger.Warnw("retry count exceeded, abandoning")
			} else {
				done = false
			}
		}
	}

	if m.remote.Epoch == runner.epoch {
		// Need one more sync to ensure remote runner list is up-to-date.
		done = false
	}

	if !done {
		return
	}

	m.logger.Infow("removing runner",
		"id", runner.instanceID,
		"runnerName", runner.runnerName,
	)
	delete(m.localRunners, runner.instanceID)
}

func (m *Monitor) handleMessage(msg any) {
	switch msg := msg.(type) {
	case MonitorMsgRegister:
		runner := &localRunner{
			instanceID: msg.InstanceID,
			instance:   msg.Instance,
		}
		runner.update(m.remote.Epoch, RunnerStatePending)

		m.logger.Infow("registering runner",
			"id", runner.instanceID,
		)
		m.localRunners[runner.instanceID] = runner

	case MonitorMsgUpdate:
		runner := m.localRunners[msg.InstanceID]
		if runner.runnerName != msg.RunnerName && msg.RunnerName != "" {
			m.logger.Infow("configuring runner",
				"id", runner.instanceID,
				"runnerName", msg.RunnerName,
			)

			runner.runnerName = msg.RunnerName
			runner.update(m.remote.Epoch, RunnerStateConfiguring)
		}

		if runner.runnerID != msg.RunnerID && msg.RunnerID != 0 {
			m.logger.Infow("starting runner",
				"id", runner.instanceID,
				"runnerName", runner.runnerName,
				"runnerID", msg.RunnerID,
			)

			runner.runnerID = msg.RunnerID
			runner.update(m.remote.Epoch, RunnerStateStarting)
		}

	case MonitorMsgExited:
		runner := m.localRunners[msg.InstanceID]
		m.logger.Infow("terminating runner",
			"id", runner.instanceID,
			"runnerName", runner.runnerName,
			"runnerID", runner.runnerID,
		)

		runner.update(m.remote.Epoch, RunnerStateTerminating)
		runner.isDead = true
		m.terminate(runner)
	}
}

func (m *Monitor) checkTimeout(runner *localRunner) bool {
	if (m.remote.Epoch - runner.epoch) > transitionTimeoutEpochs {
		m.logger.Warnw("runner timed out, terminating",
			"id", runner.instanceID,
			"runnerName", runner.runnerName,
			"elapsed", m.remote.BeginTime.Sub(runner.lastTransitionTime).String(),
		)

		runner.update(m.remote.Epoch, RunnerStateTerminating)
		runner.instance.Terminate(true)
		m.terminate(runner)
		return false
	}
	return true
}

func (m *Monitor) checkRunners() {
	m.logger.Infow("checking runners",
		"count", len(m.localRunners),
	)

	for _, runner := range m.localRunners {
		switch runner.state {
		case RunnerStatePending:
			m.checkTimeout(runner)

		case RunnerStateConfiguring:
			m.checkTimeout(runner)

		case RunnerStateStarting:
			if !m.checkTimeout(runner) {
				break
			}

			if r, ok := m.remote.Lookup(runner.runnerName, runner.runnerID); ok && r.IsOnline {
				m.logger.Infow("runner is ready",
					"id", runner.instanceID,
					"runnerName", runner.runnerName,
				)
				runner.update(m.remote.Epoch, RunnerStateReady)
			}

		case RunnerStateReady:
			if r, ok := m.remote.Lookup(runner.runnerName, runner.runnerID); !ok || !r.IsOnline {
				m.logger.Infow("runner is gone",
					"id", runner.instanceID,
					"runnerName", runner.runnerName,
					"found", ok,
					"online", ok && r.IsOnline,
				)

				runner.update(m.remote.Epoch, RunnerStateTerminating)
				m.terminate(runner)
			}

		case RunnerStateTerminating:
			m.terminate(runner)
		}
	}
}

func (m *Monitor) cleanupRunners() {
	m.logger.Info("cleaning up runners")
	for _, runner := range m.localRunners {
		runner.update(m.remote.Epoch, RunnerStateTerminating)
		m.terminate(runner)
	}
}
