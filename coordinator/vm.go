package main

import (
	"context"
	"os/exec"

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

func (v *VM) Start(ctx context.Context) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, v.VMCtlPath, "start", "--config", v.ConfigPath, "--bundle", v.BundlePath)
	v.logger.Debugw("starting vm", "cmd", cmd.String())
	return cmd, cmd.Start()
}
