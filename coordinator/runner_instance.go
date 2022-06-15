package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"

	"go.uber.org/zap"
)

type RunnerInstance struct {
	logger     *zap.SugaredLogger
	vmctlPath  string
	bundlePath string
	Config     *RunnerConfig

	ID         uint32
	Token      string
	runnerID   int
	runnerName string
	hostName   string

	Messages chan<- any
	messages <-chan any
}

var nextID uint32 = 0

func NewRunnerInstance(logger *zap.SugaredLogger, vmctlPath, bundlePath string, config *RunnerConfig) *RunnerInstance {
	id := atomic.AddUint32(&nextID, 1)
	messages := make(chan any)
	return &RunnerInstance{
		ID:         id,
		logger:     logger.Named(fmt.Sprintf("vm-%d", id)),
		vmctlPath:  vmctlPath,
		bundlePath: bundlePath,
		Config:     config,
		Messages:   messages,
		messages:   messages,
	}
}

func (r *RunnerInstance) vmctl(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, r.vmctlPath, args...)
}

func (r *RunnerInstance) Init(ctx context.Context) error {
	cmd := r.vmctl(ctx, "clone", r.Config.BaseVMBundlePath, r.bundlePath)
	r.logger.Debugw("cloning vm", "cmd", cmd.String())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone VM: %w", err)
	}

	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	r.Token = fmt.Sprintf("%s-%d", base64.RawURLEncoding.EncodeToString(buf[:]), r.ID)
	r.logger.Infow("generated token", "mac", r.Token)

	return nil
}

func (r *RunnerInstance) start(ctx context.Context) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
	cmd := r.vmctl(ctx, "start", "--config", r.Config.VMConfigPath, "--bundle", r.bundlePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot setup out pipe: %w", err)
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	defer pw.Close()

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot setup in pipe: %w", err)
	}

	r.logger.Debugw("starting vm", "cmd", cmd.String())
	err = cmd.Start()
	if err != nil {
		pr.Close()
		return nil, nil, nil, err
	}

	return cmd, in, pr, nil
}

func (r *RunnerInstance) Run(ctx context.Context) (bool, error) {
	cmd, in, out, err := r.start(context.Background())
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

	in.Write([]byte(r.Token + "\n"))

	completed := make(chan error, 1)
	go func() {
		completed <- cmd.Wait()
	}()

	for {
		select {
		case msg := <-r.messages:
			r.handleMessage(msg)

		case err = <-completed:
			return true, err

		case <-ctx.Done():
			r.logger.Infow("terminating VM")
			return false, cmd.Process.Kill()
		}
	}
}

func (r *RunnerInstance) handleMessage(msg any) {
	if reg, ok := msg.(RunnerMsgRegister); ok {
		r.logger.Infow("registering instance",
			"name", reg.Name,
			"hostname", reg.HostName)
		r.runnerName = reg.Name
		r.hostName = reg.HostName
	} else if reg, ok := msg.(RunnerMsgUpdate); ok {
		r.logger.Infow("updating instance", "runnerID", reg.RunnerID)
		if reg.RunnerID != nil {
			r.runnerID = *reg.RunnerID
		}
	}
}
