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

func (t *RunnerTargetRepository) GetRunners(
	ctx context.Context, client *github.Client, page int, pageSize int,
) ([]*github.Runner, int, error) {
	runners, resp, err := client.Actions.ListRunners(ctx, t.Owner, t.Name, &github.ListOptions{Page: page, PerPage: pageSize})
	if err != nil {
		return nil, 0, err
	}

	return runners.Runners, resp.NextPage, nil
}

func (t *RunnerTargetRepository) DeleteRunner(
	ctx context.Context, client *github.Client, id int64,
) error {
	_, err := client.Actions.RemoveRunner(ctx, t.Owner, t.Name, id)
	return err
}
