package main

type RunnerMsgRegister struct {
	Name     string
	HostName string
}

type RunnerMsgUpdate struct {
	RunnerID *int
}
