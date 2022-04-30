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

// AuthConfig is authentication configuration in RepositoryConfig.
type AuthConfig struct {
	Username                      string `json:"username,omitempty"`
	Password                      string `json:"password,omitempty"`
	KeyPath                       string `json:"key_path,omitempty"`
	KeyPassphrase                 string `json:"key_passphrase,omitempty"`
	StrictHostKeyCheckingDisabled bool   `json:"strict_host_key_checking_disabled,omitempty"`
}

// WebhookConfig is a webhook configuration in RepositoryConfig.
type WebhookConfig struct {
	Name   string `json:"name,omitempty"`
	Header string `json:"header,omitempty"`
	Secret string `json:"secret,omitempty"`
}

// ExecConfig is an execution script configuration in RepositoryConfig.
type ExecConfig struct {
	Name    string   `json:"name,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
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
	// The interval at which repository updates automatically.
	UpdateInterval int              `json:"update_interval,omitempty"`
	Auth           *AuthConfig      `json:"auth,omitempty"`
	Webhooks       []*WebhookConfig `json:"webhooks,omitempty"`
	PostPullExec   []*ExecConfig    `json:"post_pull_exec,omitempty"`
	transport      string           `json:"transport,omitempty"`
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
	if !strings.HasSuffix(rc.Address, ".git") {
		return errors.ErrRepositoryConfigAddressUnsupported.WithArgs(rc.Address)
	}

	switch {
	case strings.HasPrefix(rc.Address, "https://"), strings.HasPrefix(rc.Address, "http://"):
		rc.transport = "http"
	default:
		rc.transport = "ssh"
	}
	return nil
}
