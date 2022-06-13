package githublib

import (
	"context"
	"fmt"

	"github.com/google/go-github/v45/github"
)

type RunnerTargetRepository struct {
	Name  string
	Owner string
}

func (t *RunnerTargetRepository) URL() string {
	return fmt.Sprintf("https://github.com/%s/%s", t.Owner, t.Name)
}

func (t *RunnerTargetRepository) GetRegistrationToken(ctx context.Context, client *github.Client) (*github.RegistrationToken, error) {
	token, _, err := client.Actions.CreateRegistrationToken(ctx, t.Owner, t.Name)
	if err != nil {
		return nil, err
	}

	return token, nil
}
