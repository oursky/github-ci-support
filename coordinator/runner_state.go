package main

import (
	"bufio"
	"context"

	"go.uber.org/zap"
)

type RunnerState struct {
	Logger     *zap.SugaredLogger
	MacAddress string
	VM         *VM
}

func (s *RunnerState) Run(ctx context.Context) (bool, error) {
	cmd, out, err := s.VM.Start(context.Background())
	if err != nil {
		return true, err
	}

	go func() {
		log := s.Logger.Named("vm.log")
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
			s.Logger.Infow("terminating VM")
			return false, cmd.Process.Kill()
		}
	}
}
