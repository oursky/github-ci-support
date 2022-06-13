package githublib

import (
	"context"
	"sync"
	"time"

	"github.com/google/go-github/v45/github"
)

type RegistrationTokenStore struct {
	Target RunnerTarget
	Client *github.Client

	token *RegistrationToken
	lock  *sync.RWMutex
}

func NewRegistrationTokenStore(target RunnerTarget, client *github.Client) *RegistrationTokenStore {
	return &RegistrationTokenStore{
		Target: target,
		Client: client,
		token:  nil,
		lock:   new(sync.RWMutex),
	}
}

func (s *RegistrationTokenStore) Get() (*RegistrationToken, error) {
	token := s.read()
	if !token.needRenewal() {
		return token, nil
	}

	token, err := s.fetch()
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *RegistrationTokenStore) read() *RegistrationToken {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.token
}

func (s *RegistrationTokenStore) fetch() (*RegistrationToken, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.token.needRenewal() {
		return s.token, nil
	}

	token, err := s.Target.GetRegistrationToken(context.TODO(), s.Client)
	if err != nil {
		return nil, err
	}

	s.token = &RegistrationToken{
		Value:     token.GetToken(),
		ExpiresAt: token.GetExpiresAt().Time,
	}
	return s.token, nil
}

const renewTokenThreshold time.Duration = 60 * time.Second

type RegistrationToken struct {
	Value     string
	ExpiresAt time.Time
}

func (t *RegistrationToken) needRenewal() bool {
	return t == nil || t.ExpiresAt.Add(-renewTokenThreshold).Before(time.Now())
}
