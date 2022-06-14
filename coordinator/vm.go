package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"go.uber.org/zap"
)

type VM struct {
	logger     *zap.SugaredLogger
	VMCtlPath  string
	BundlePath string
	ConfigPath string
}

func NewVM(logger *zap.SugaredLogger, vmctlPath, bundlePath, configPath string) *VM {
	return &VM{
		logger:     logger.Named("vm"),
		VMCtlPath:  vmctlPath,
		BundlePath: bundlePath,
		ConfigPath: configPath,
	}
}

func (v *VM) CloneFrom(ctx context.Context, bundlePath string) error {
	cmd := exec.CommandContext(ctx, v.VMCtlPath, "clone", bundlePath, v.BundlePath)
	v.logger.Debugw("cloning vm", "cmd", cmd.String())
	return cmd.Run()
}

func (v *VM) Start(ctx context.Context) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, v.VMCtlPath, "start", "--config", v.ConfigPath, "--bundle", v.BundlePath)
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

	v.logger.Debugw("starting vm", "cmd", cmd.String())
	err = cmd.Start()
	if err != nil {
		pr.Close()
		return nil, nil, err
	}

	return cmd, pr, nil
}
