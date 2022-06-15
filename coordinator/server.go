package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/go-github/v45/github"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/oursky/github-ci-support/githublib"
)

const (
	serviceType = "_github-action._tcp"
	serviceName = "coordinator"
)

type Server struct {
	logger *zap.SugaredLogger
	target githublib.RunnerTarget
	client *github.Client
	token  *githublib.RegistrationTokenStore

	Instances *sync.Map
}

func NewServer(logger *zap.SugaredLogger, target githublib.RunnerTarget, client *github.Client) *Server {
	return &Server{
		logger:    logger.Named("server"),
		target:    target,
		client:    client,
		token:     githublib.NewRegistrationTokenStore(target, client),
		Instances: new(sync.Map),
	}
}

func (s *Server) Run(ctx context.Context, g *errgroup.Group) {
	g.Go(func() error {
		listener, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			return fmt.Errorf("cannot setup server listener: %w", err)
		}

		g.Go(func() error {
			return s.runMDNS(ctx, listener)
		})
		s.runHTTP(ctx, listener)
		return nil
	})
}

func (s *Server) runMDNS(ctx context.Context, listener net.Listener) error {
	addr := listener.Addr().(*net.TCPAddr)
	cmd := exec.Command("dns-sd", "-R", serviceName, serviceType, "local", strconv.Itoa(addr.Port))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("cannot publish mDNS: %w", err)
	}

	s.logger.Infow("published mDNS service",
		"service", fmt.Sprintf("%s.%s.local", serviceName, serviceType),
		"port", addr.Port,
	)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		cmd.Process.Signal(syscall.SIGTERM)
		return nil
	}
}

func (s *Server) runHTTP(ctx context.Context, listener net.Listener) {
	mux := http.NewServeMux()
	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 100 * time.Second,
		Handler:      mux,
		ErrorLog:     zap.NewStdLog(s.logger.Desugar()),
	}

	mux.HandleFunc("/register", s.register)
	mux.HandleFunc("/update", s.update)
	mux.HandleFunc("/wait", s.wait)

	addr := listener.Addr().(*net.TCPAddr)
	s.logger.Infow("server started", "addr", addr.String())

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(context.Background()); err != nil {
			s.logger.Errorw("failed to shutdown server", "error", err)
		}
	}()

	err := server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Errorw("failed to start server", "error", err)
	}
}

func (s *Server) register(rw http.ResponseWriter, r *http.Request) {
	instance, ok := s.check(rw, r, true)
	if !ok {
		return
	}

	name := r.FormValue("name")
	hostName := r.FormValue("hostName")
	instance.Post(RunnerMsgRegister{Name: name, HostName: hostName})

	token, err := s.token.Get()
	if err != nil {
		s.logger.Errorw("cannot get registration token", "error", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	type resp struct {
		Name      string `json:"name"`
		GitHubURL string `json:"gitHubURL"`
		Token     string `json:"token"`
		Group     string `json:"group"`
		Labels    string `json:"labels"`
	}
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(resp{
		Name:      name,
		GitHubURL: s.target.URL(),
		Token:     token.Value,
		Group:     instance.Config.RunnerGroup,
		Labels:    strings.Join(instance.Config.Labels, ","),
	})
}

func (s *Server) update(rw http.ResponseWriter, r *http.Request) {
	instance, ok := s.check(rw, r, true)
	if !ok {
		return
	}

	runnerIDStr := r.FormValue("runnerID")
	var runnerID *int64
	if runnerIDStr != "" {
		id, err := strconv.ParseInt(runnerIDStr, 10, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(err.Error()))
			return
		}
		runnerID = &id
	}
	instance.Post(RunnerMsgUpdate{RunnerID: runnerID})

	rw.WriteHeader(http.StatusNoContent)
}

func (s *Server) wait(rw http.ResponseWriter, r *http.Request) {
	instance, ok := s.check(rw, r, false)
	if !ok {
		return
	}

	select {
	case <-instance.NeedTerminate():
		rw.WriteHeader(http.StatusNoContent)
	case <-time.After(60 * time.Second):
		rw.WriteHeader(http.StatusRequestTimeout)
	}
}

func (s *Server) check(rw http.ResponseWriter, r *http.Request, parseForm bool) (*RunnerInstance, bool) {
	authz := r.Header.Get("Authorization")
	bearer, token, ok := strings.Cut(authz, " ")
	if !ok || bearer != "Bearer" {
		s.reqError(rw, "invalid authz header")
		return nil, false
	}

	instance, ok := s.Instances.Load(token)
	if !ok {
		s.reqError(rw, "invalid token")
		return nil, false
	}

	if parseForm {
		err := r.ParseForm()
		if err != nil {
			err = fmt.Errorf("malformed request: %w", err)
			s.reqError(rw, err.Error())
			return nil, false
		}
	}

	return instance.(*RunnerInstance), true
}

func (s *Server) reqError(rw http.ResponseWriter, msg string) {
	s.logger.Debug(msg)
	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte(msg))
}
