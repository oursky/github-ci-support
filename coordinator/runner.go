package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Runner struct {
	id        int
	logger    *zap.SugaredLogger
	vmctlPath string
	config    *RunnerConfig
	server    *Server
	monitor   *Monitor
}

func NewRunner(id int, logger *zap.SugaredLogger, config *Config, runnerConfig RunnerConfig, server *Server, monitor *Monitor) *Runner {
	return &Runner{
		id:        id,
		logger:    logger.Named(fmt.Sprintf("runner-%d", id)),
		vmctlPath: config.VMCtlPath,
		config:    &runnerConfig,
		server:    server,
		monitor:   monitor,
	}
}

func (r *Runner) Run(ctx context.Context, g *errgroup.Group, serverPort int) {
	g.Go(func() error {
		return r.run(ctx, serverPort)
	})
}

func (r *Runner) run(ctx context.Context, serverPort int) error {
	workDir, err := os.MkdirTemp("", fmt.Sprintf("runner-%d-*", r.id))
	if err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	r.logger.Infow("created working directory", "dir", workDir)

	defer func() {
		r.logger.Infow("deleting working directory", "dir", workDir)
		os.RemoveAll(workDir)
	}()

	bundlePath := filepath.Join(workDir, "vm.bundle")

	for ctx.Err() == nil {
		err = r.runVM(ctx, bundlePath, serverPort)
		if err != nil {
			return fmt.Errorf("failed to run VM: %w", err)
		}
		r.logger.Info("VM exited, restarting VM")
	}

	return nil
}

func (r *Runner) runVM(ctx context.Context, bundlePath string, serverPort int) error {
	instance := NewRunnerInstance(r.logger, r.vmctlPath, bundlePath, r.config, r.monitor, serverPort)

	err := instance.Init(ctx)
	if err != nil {
		return fmt.Errorf("failed to init VM: %w", err)
	}

	r.server.Instances.Store(instance.Token, instance)
	defer r.server.Instances.Delete(instance.Token)

	return instance.Run(ctx)
}
