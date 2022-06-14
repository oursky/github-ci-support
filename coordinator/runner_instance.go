package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"go.uber.org/zap"
)

type RunnerInstance struct {
	logger     *zap.SugaredLogger
	MacAddress string
	vmctlPath  string
	bundlePath string
	configPath string
}

func NewRunnerInstance(logger *zap.SugaredLogger, vmctlPath, bundlePath, configPath string) *RunnerInstance {
	return &RunnerInstance{
		logger:     logger.Named("vm"),
		vmctlPath:  vmctlPath,
		bundlePath: bundlePath,
		configPath: configPath,
	}
}

func (r *RunnerInstance) vmctl(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, r.vmctlPath, args...)
}

func (r *RunnerInstance) Init(ctx context.Context, baseBundlePath, baseConfigPath string) error {
	cmd := r.vmctl(ctx, "clone", baseBundlePath, r.bundlePath)
	r.logger.Debugw("cloning vm", "cmd", cmd.String())
	if err := cmd.Run(); err != nil {
		return err
	}

	r.MacAddress = generateMACAddress()
	r.logger.Infow("generated MAC address", "mac", r.MacAddress)

	configData, err := ioutil.ReadFile(baseConfigPath)
	if err != nil {
		return fmt.Errorf("failed load VM config: %w", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed parse VM config: %w", err)
	}

	config["macAddress"] = r.MacAddress

	configData, err = json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed serialize VM config: %w", err)
	}
	if err := ioutil.WriteFile(r.configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed save VM config: %w", err)
	}

	return nil
}

func (r *RunnerInstance) start(ctx context.Context) (*exec.Cmd, io.ReadCloser, error) {
	cmd := r.vmctl(ctx, "start", "--config", r.configPath, "--bundle", r.bundlePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot setup pipe: %w", err)
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	defer pw.Close()

	r.logger.Debugw("starting vm", "cmd", cmd.String())
	err = cmd.Start()
	if err != nil {
		pr.Close()
		return nil, nil, err
	}

	return cmd, pr, nil
}

func (r *RunnerInstance) Run(ctx context.Context) (bool, error) {
	cmd, out, err := r.start(context.Background())
	if err != nil {
		return true, err
	}

	go func() {
		log := r.logger.Named("log")
		defer out.Close()

		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			log.Infof(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			log.Errorw("cannot scan VM output", "error", err)
		}
	}()

	completed := make(chan error, 1)
	go func() {
		completed <- cmd.Wait()
	}()

	for {
		select {
		case err = <-completed:
			return true, err

		case <-ctx.Done():
			r.logger.Infow("terminating VM")
			return false, cmd.Process.Kill()
		}
	}
}
