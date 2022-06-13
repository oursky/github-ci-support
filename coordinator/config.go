package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/oursky/github-ci-support/githublib"
)

type Config struct {
	Auth      githublib.AuthConfig `json:"auth"`
	Target    string               `json:"target"`
	Runners   []RunnerConfig       `json:"runners"`
	VMCtlPath string               `json:"vmctlPath"`
}

type RunnerConfig struct {
	BaseVMBundlePath string `json:"baseVMBundlePath"`
	VMConfigPath     string `json:"vmConfigPath"`

	RunnerGroup string   `json:"runnerGroup,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

func NewConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
