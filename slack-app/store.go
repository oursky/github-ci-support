package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type Store struct {
	ctx       context.Context
	cli       *kubernetes.Clientset
	lock      *sync.RWMutex
	config    map[string]string
	namespace string
	name      string
}

func NewStore(ctx context.Context, namespace string, name string) (*Store, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
		config, err = kubeConfig.ClientConfig()

		if err != nil {
			return nil, err
		}
	}

	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cms := cli.CoreV1().ConfigMaps(namespace)
	_, err = cms.Create(ctx, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string]string{},
	}, metav1.CreateOptions{})

	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, err
		}
	}

	return &Store{
		ctx:       ctx,
		cli:       cli,
		namespace: namespace,
		name:      name,
		lock:      new(sync.RWMutex),
		config:    map[string]string{},
	}, nil
}

func (s *Store) loadConfig(cm *v1.ConfigMap) {
	s.lock.Lock()
	defer s.lock.Unlock()

	config := map[string]string{}
	for k, v := range cm.Data {
		config[k] = v
	}
	s.config = config

	log.Printf("config loaded: %d entries", len(config))
}

func (s *Store) Start(stop <-chan struct{}) {
	watchlist := cache.NewListWatchFromClient(
		s.cli.CoreV1().RESTClient(),
		string(v1.ResourceConfigMaps),
		s.namespace,
		fields.Everything(),
	)
	_, controller := cache.NewInformer(
		watchlist,
		&v1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				cm := obj.(*v1.ConfigMap)
				if cm.Name == s.name {
					s.loadConfig(cm)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				cm := newObj.(*v1.ConfigMap)
				if cm.Name == s.name {
					s.loadConfig(cm)
				}
			},
			DeleteFunc: nil,
		},
	)

	go controller.Run(stop)
}

func (s *Store) GetChannels(repo string) []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	data, _ := s.config[repoKey(repo)]
	if data == "" {
		return nil
	}
	return strings.Split(data, ";")
}

func (s *Store) AddChannel(repo string, channelID string) error {
	channelIDs := s.GetChannels(repo)
	for _, c := range channelIDs {
		if c == channelID {
			return fmt.Errorf("already subscribed to repo")
		}
	}
	channelIDs = append(channelIDs, channelID)
	data := strings.Join(channelIDs, ";")

	_, err := s.cli.CoreV1().ConfigMaps(s.namespace).Patch(
		s.ctx,
		s.name,
		types.StrategicMergePatchType,
		makePatch(repo, data),
		metav1.PatchOptions{})
	return err
}

func (s *Store) DelChannel(repo string, channelID string) error {
	channelIDs := s.GetChannels(repo)
	newChannelIDs := []string{}
	found := false
	for _, c := range channelIDs {
		if c == channelID {
			found = true
			continue
		}
		newChannelIDs = append(newChannelIDs, c)
	}
	if !found {
		return fmt.Errorf("not subscribed to repo")
	}
	data := strings.Join(newChannelIDs, ";")

	_, err := s.cli.CoreV1().ConfigMaps(s.namespace).Patch(
		s.ctx,
		s.name,
		types.StrategicMergePatchType,
		makePatch(repo, data),
		metav1.PatchOptions{})
	return err
}

func repoKey(repo string) string {
	return strings.ReplaceAll(repo, "/", ".")
}

func makePatch(repo string, data string) []byte {
	type patch struct {
		Data map[string]string `json:"data"`
	}
	json, err := json.Marshal(patch{Data: map[string]string{
		repoKey(repo): data,
	}})
	if err != nil {
		panic(err)
	}
	return json
}
