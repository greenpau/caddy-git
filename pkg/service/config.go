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
	"github.com/greenpau/caddy-git/pkg/errors"
	"strings"
)

// Config is a configuration of Manager.
type Config struct {
	Repositories []*RepositoryConfig `json:"repositories,omitempty"`
	repoMap      map[string]*RepositoryConfig
}

// RepositoryConfig is a configuration of Repository.
type RepositoryConfig struct {
	// The alias for the Repository.
	Name string `json:"name,omitempty"`
	// The address of the Repository.
	Address string `json:"address,omitempty"`
	// The directory where the Repository is being stored locally.
	BaseDir string `json:"base_dir,omitempty"`
	Branch  string `json:"branch,omitempty"`
	Depth   int    `json:"depth,omitempty"`
}

// NewConfig returns an instance of Config.
func NewConfig() *Config {
	return &Config{
		repoMap: make(map[string]*RepositoryConfig),
	}
}

// NewRepositoryConfig returns an instance of RepositoryConfig.
func NewRepositoryConfig() *RepositoryConfig {
	return &RepositoryConfig{}
}

// AddRepository adds a repository entry to Config.
func (cfg *Config) AddRepository(rc *RepositoryConfig) error {

	if rc == nil {
		return errors.ErrRepositoryConfigNil
	}
	rc.Name = strings.TrimSpace(rc.Name)
	if rc.Name == "" {
		return errors.ErrRepositoryConfigNameEmpty
	}
	if _, exists := cfg.repoMap[rc.Name]; exists {
		return errors.ErrRepositoryConfigExists.WithArgs(rc.Name)
	}
	if err := rc.validate(); err != nil {
		return err
	}
	cfg.Repositories = append(cfg.Repositories, rc)
	cfg.repoMap[rc.Name] = rc
	return nil
}

func (rc *RepositoryConfig) validate() error {
	if rc.Address == "" {
		return errors.ErrRepositoryConfigAddressEmpty
	}
	switch {
	case strings.HasPrefix(rc.Address, "http"):
	default:
		return errors.ErrRepositoryConfigAddressUnsupported.WithArgs(rc.Address)
	}
	return nil
}
