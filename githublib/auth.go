package githublib

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"golang.org/x/oauth2"
)

type AuthType string

const (
	AuthTypeToken AuthType = "Token"
	AuthTypeApp   AuthType = "App"
)

type AuthConfig struct {
	Type  AuthType       `json:"type"`
	Token string         `json:"token,omitempty"`
	App   *AppAuthConfig `json:"app,omitempty"`
}

type AppAuthConfig struct {
	AppID          int64  `json:"appID"`
	InstallationID int64  `json:"installationID"`
	PrivateKeyPath string `json:"privateKeyPath"`
}

func (c *AuthConfig) CreateClient() (*http.Client, error) {
	var transport http.RoundTripper
	switch c.Type {
	case AuthTypeToken:
		if c.Token == "" {
			return nil, fmt.Errorf("missing GitHub token")
		}

		transport = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.Token},
		)).Transport

	case AuthTypeApp:
		if c.App == nil {
			return nil, fmt.Errorf("missing GitHub app key")
		}

		itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, c.App.AppID, c.App.InstallationID, c.App.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load app key: %w", err)
		}
		transport = itr

	default:
		return nil, fmt.Errorf("invalid auth type: %s", c.Type)
	}

	client := &http.Client{Transport: transport}
	client.Timeout = 10 * time.Second

	return &http.Client{Transport: transport}, nil
}
