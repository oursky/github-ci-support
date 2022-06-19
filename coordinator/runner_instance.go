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
	"sync"
	"sync/atomic"
	"syscall"

	"go.uber.org/zap"
)

type RunnerInstance struct {
	logger     *zap.SugaredLogger
	vmctlPath  string
	bundlePath string
	Config     *RunnerConfig
	monitor    *Monitor
	serverPort int
	serverURL  string

	id         uint32
	Token      string
	runnerID   int64
	runnerName string
	hostName   string

	termLock  *sync.Mutex
	term      int
	terminate chan struct{}
	kill      chan struct{}
	messages  chan any
}

var nextID uint32 = 0

func NewRunnerInstance(logger *zap.SugaredLogger, vmctlPath, bundlePath string, config *RunnerConfig, monitor *Monitor, serverPort int) *RunnerInstance {
	id := atomic.AddUint32(&nextID, 1)
	return &RunnerInstance{
		id:         id,
		logger:     logger.Named(fmt.Sprintf("vm-%d", id)),
		vmctlPath:  vmctlPath,
		bundlePath: bundlePath,
		Config:     config,
		monitor:    monitor,
		serverPort: serverPort,
		serverURL:  "",
		termLock:   new(sync.Mutex),
		term:       0,
		terminate:  make(chan struct{}),
		kill:       make(chan struct{}),
		messages:   make(chan any),
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
	r.Token = fmt.Sprintf("%s-%d", base64.RawURLEncoding.EncodeToString(buf[:]), r.id)
	r.logger.Infow("generated token", "mac", r.Token)

	hostName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("cannot get hostname: %w", err)
	}
	r.serverURL = fmt.Sprintf("http://%s:%d", hostName, r.serverPort)

	return nil
}

func (r *RunnerInstance) Post(msg any) {
	select {
	case <-r.terminate:
	case r.messages <- msg:
	}
}

func (r *RunnerInstance) Terminate(kill bool) {
	r.termLock.Lock()
	defer r.termLock.Unlock()
	if r.term < 1 {
		r.term = 1
		close(r.terminate)
	}
	if r.term < 2 && kill {
		r.term = 2
		close(r.kill)
	}
}

func (r *RunnerInstance) NeedTerminate() <-chan struct{} {
	return r.terminate
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

func (r *RunnerInstance) Run(ctx context.Context) error {
	cmd, in, out, err := r.start(context.Background())
	if err != nil {
		return err
	}

	r.monitor.Post(MonitorMsgRegister{InstanceID: r.id, Instance: r})
	defer r.monitor.Post(MonitorMsgExited{InstanceID: r.id})

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

	bootstrapMsg := fmt.Sprintf("%s %s\n", r.serverURL, r.Token)
	in.Write([]byte(bootstrapMsg))

	completed := make(chan error, 1)
	go func() {
		completed <- cmd.Wait()
	}()

	terminate := false
	for !terminate {
		select {
		case msg := <-r.messages:
			r.handleMessage(msg)

		case err = <-completed:
			return err

		case <-ctx.Done():
			terminate = true

		case <-r.terminate:
			terminate = true
		}
	}

	r.logger.Infow("terminating VM gracefully")
	select {
	case err = <-completed:
		return err
	case <-r.kill:
		r.logger.Infow("killing VM")
		return cmd.Process.Kill()
	}
}

func (r *RunnerInstance) handleMessage(msg any) {
	switch msg := msg.(type) {
	case RunnerMsgRegister:
		r.runnerName = msg.Name
		r.hostName = msg.HostName

	case RunnerMsgUpdate:
		if msg.RunnerID != nil {
			r.runnerID = *msg.RunnerID
		}
	}

	r.monitor.Post(MonitorMsgUpdate{
		InstanceID: r.id,
		RunnerName: r.runnerName,
		RunnerID:   r.runnerID,
	})
}
