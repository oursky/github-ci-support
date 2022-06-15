package main

type MonitorMsgRegister struct {
	InstanceID uint32
	Instance   *RunnerInstance
}

type MonitorMsgUpdate struct {
	InstanceID uint32
	RunnerName string
	RunnerID   int64
}

type MonitorMsgExited struct {
	InstanceID uint32
	RunnerName string
}
