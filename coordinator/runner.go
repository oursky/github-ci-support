package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/oursky/github-ci-support/githublib"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Runner struct {
	id        int
	logger    *zap.SugaredLogger
	vmctlPath string
	config    *RunnerConfig
	token     *githublib.RegistrationTokenStore
}

func NewRunner(id int, logger *zap.SugaredLogger, config *Config, runnerConfig RunnerConfig, token *githublib.RegistrationTokenStore) *Runner {
	return &Runner{
		id:        id,
		logger:    logger.Named(fmt.Sprintf("runner-%d", id)),
		vmctlPath: config.VMCtlPath,
		config:    &runnerConfig,
		token:     token,
	}
}

func (r *Runner) Run(ctx context.Context, g *errgroup.Group) {
	g.Go(func() error {
		return r.run(ctx)
	})
}

func (r *Runner) run(ctx context.Context) error {
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
	configPath := filepath.Join(workDir, "config.json")

	cont := true
	for cont {
		cont, err = r.runVM(ctx, bundlePath, configPath)
		if err != nil {
			return fmt.Errorf("failed to run VM: %w", err)
		}
		if cont {
			r.logger.Info("VM exited, restarting VM")
		}
	}

	return nil
}

func (r *Runner) runVM(ctx context.Context, bundlePath, configPath string) (bool, error) {
	vm := NewVM(r.logger, r.vmctlPath, bundlePath, configPath)

	r.logger.Infow("cloning VM", "from", r.config.BaseVMBundlePath, "to", bundlePath)
	err := vm.CloneFrom(ctx, r.config.BaseVMBundlePath)
	if err != nil {
		return false, fmt.Errorf("failed to clone VM: %w", err)
	}

	configData, err := ioutil.ReadFile(r.config.VMConfigPath)
	if err != nil {
		return false, fmt.Errorf("failed load VM config: %w", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return false, fmt.Errorf("failed parse VM config: %w", err)
	}

	macAddr := generateMACAddress()
	r.logger.Infow("generated MAC address", "mac", macAddr)
	config["macAddress"] = macAddr

	configData, err = json.Marshal(config)
	if err != nil {
		return false, fmt.Errorf("failed serialize VM config: %w", err)
	}
	if err := ioutil.WriteFile(configPath, configData, 0644); err != nil {
		return false, fmt.Errorf("failed save VM config: %w", err)
	}

	state := RunnerState{Logger: r.logger, VM: vm, MacAddress: macAddr}
	cont, err := state.Run(ctx)
	if err != nil {
		return cont, err
	}

	return cont, nil
}
