// Copyright 2022 Paul Greenberg greenpau@outlook.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"go.uber.org/zap"
	"sync"
)

var manager *Manager

// Manager manages git repositories
type Manager struct {
	mu      sync.Mutex
	repos   map[string]*Repository
	started bool
	logger  *zap.Logger
}

// NewManager parses config and creates Manager instance.
func NewManager(cfg *Config, logger *zap.Logger) (*Manager, error) {
	m := &Manager{
		repos:  make(map[string]*Repository),
		logger: logger,
	}
	manager = m
	for _, rc := range cfg.Repositories {
		if err := rc.validate(); err != nil {
			return nil, err
		}
		r, _ := NewRepository(rc)
		r.logger = logger
		m.repos[rc.Name] = r
		if err := r.update(); err != nil {
			m.logger.Error("failed managing repo", zap.String("repo_name", rc.Name), zap.Error(err))
			return nil, err
		}
		m.logger.Debug("registered and synced repo", zap.String("repo_name", rc.Name))
		if rc.UpdateInterval > 0 {
			go autoUpdater(r)
		}
	}
	return m, nil
}

// Start starts Manager.
func (m *Manager) Start() []*Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

// Stop stops Manager.
func (m *Manager) Stop() []*Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}
