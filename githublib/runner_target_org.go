package githublib

import (
	"context"
	"fmt"

	"github.com/google/go-github/v45/github"
)

type RunnerTargetOrganization struct {
	Name string
}

func (t *RunnerTargetOrganization) URL() string {
	return fmt.Sprintf("https://github.com/%s", t.Name)
}

func (t *RunnerTargetOrganization) GetRegistrationToken(ctx context.Context, client *github.Client) (*github.RegistrationToken, error) {
	token, _, err := client.Actions.CreateOrganizationRegistrationToken(ctx, t.Name)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (t *RunnerTargetOrganization) GetRunners(
	ctx context.Context, client *github.Client, page int, pageSize int,
) ([]*github.Runner, int, error) {
	runners, resp, err := client.Actions.ListOrganizationRunners(ctx, t.Name, &github.ListOptions{Page: page, PerPage: pageSize})
	if err != nil {
		return nil, 0, err
	}

	return runners.Runners, resp.NextPage, nil
}

func (t *RunnerTargetOrganization) DeleteRunner(
	ctx context.Context, client *github.Client, id int64,
) error {
	_, err := client.Actions.RemoveOrganizationRunner(ctx, t.Name, id)
	return err
}
