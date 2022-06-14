package githublib

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/go-github/v45/github"
)

type RunnerTarget interface {
	URL() string
	GetRegistrationToken(ctx context.Context, client *github.Client) (*github.RegistrationToken, error)
}

var (
	regexTargetRepo = regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)/?`)
)

func NewRunnerTarget(url string) (RunnerTarget, error) {
	if match := regexTargetRepo.FindStringSubmatch(url); match != nil {
		owner := match[1]
		name := match[2]
		return &RunnerTargetRepository{Name: name, Owner: owner}, nil
	}
	return nil, errors.New("unsupported GitHub URL")
}