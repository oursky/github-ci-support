package githublib

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

type AuthType string

const (
	AuthTypeToken AuthType = "Token"
)

type AuthConfig struct {
	Type  AuthType `json:"type"`
	Token string   `json:"token"`
}

func (c *AuthConfig) CreateClient() (*github.Client, error) {
	var transport http.RoundTripper
	switch c.Type {
	case AuthTypeToken:
		transport = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.Token},
		)).Transport

	default:
		return nil, fmt.Errorf("invalid auth type: %s", c.Type)
	}

	return github.NewClient(&http.Client{Transport: transport}), nil
}
