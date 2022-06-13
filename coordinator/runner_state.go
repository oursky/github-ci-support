package main

import "context"

type RunnerState struct {
	MacAddress string
	VM         *VM
}

func (s *RunnerState) Run(ctx context.Context) error {
	cmd, err := s.VM.Start(ctx)
	if err != nil {
		return err
	}

	completed := make(chan error, 1)
	go func() {
		completed <- cmd.Wait()
	}()

	err = <-completed
	return err
}
