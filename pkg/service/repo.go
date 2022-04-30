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
	"bytes"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"go.uber.org/zap"
	cryptossh "golang.org/x/crypto/ssh"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Repository is a configuration for a command or app.
type Repository struct {
	Config      *RepositoryConfig `json:"config,omitempty"`
	mu          sync.Mutex
	logger      *zap.Logger
	lastUpdated time.Time
	updating    bool
}

// NewRepository returns an instance of Repository.
func NewRepository(rc *RepositoryConfig) (*Repository, error) {
	r := &Repository{
		Config: rc,
	}
	return r, nil
}

func (r *Repository) update() error {
	if r.updating {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.updating = true
	defer func() {
		r.updating = false
	}()

	err := r.runUpdate()
	if err != nil {
		return err
	}

	if len(r.Config.PostPullExec) > 0 {
		r.runPostPullExec()
	}

	return nil
}

func (r *Repository) runPostPullExec() {
	for _, entry := range r.Config.PostPullExec {
		var stdout, stderr bytes.Buffer
		switch {
		case entry.Command != "":
			cmd := exec.Command(entry.Command, entry.Args...)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				r.logger.Warn(
					"failed executing post-pull command",
					zap.String("repo_name", r.Config.Name),
					zap.String("error", fmt.Sprintf("%v", cmd.Stderr)),
				)
				continue
			}
			r.logger.Debug(
				"executed post-pull command",
				zap.String("repo_name", r.Config.Name),
				zap.String("stdout", fmt.Sprintf("%v", cmd.Stdout)),
				zap.String("stderr", fmt.Sprintf("%v", cmd.Stderr)),
			)
		}
	}
}

func (r *Repository) runUpdate() error {
	r.Config.BaseDir = expandDir(r.Config.BaseDir)

	baseDirExists, err := dirExists(r.Config.BaseDir)
	if err != nil {
		return err
	}
	if !baseDirExists {
		if err := os.MkdirAll(r.Config.BaseDir, 0700); err != nil {
			return err
		}
	}

	repoDir := path.Join(r.Config.BaseDir, r.Config.Name)
	repoDirExists, err := dirExists(repoDir)
	if err != nil {
		return err
	}
	if !repoDirExists {
		// Clone the repository.
		opts := &git.CloneOptions{}
		if err := configureCloneOptions(r.Config, opts); err != nil {
			return err
		}
		if _, err := git.PlainClone(repoDir, false, opts); err != nil {
			return err
		}
	}

	// Pull the repository.
	repoDir, err = filepath.Abs(repoDir)
	if err != nil {
		return err
	}
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	opts := &git.PullOptions{}
	if err := configurePullOptions(r.Config, opts); err != nil {
		return err
	}
	if err := w.Pull(opts); err != nil {
		if err == git.NoErrAlreadyUpToDate {
			r.logger.Debug(
				"repo is already up to date",
				zap.String("repo_name", r.Config.Name),
			)
			return nil
		}
		return err
	}
	ref, err := repo.Head()
	if err != nil {
		return err
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	r.logger.Debug(
		"pulled latest commit",
		zap.String("repo_name", r.Config.Name),
		zap.Any("commit", commit.Hash.String()),
	)
	return nil
}

func dirExists(s string) (bool, error) {
	if s == "" {
		return true, nil
	}
	_, err := os.Stat(s)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err

}

func configureCloneOptions(cfg *RepositoryConfig, opts *git.CloneOptions) error {
	opts.URL = cfg.Address
	trAuthMethod, err := configureAuthOptions(cfg)
	if err != nil {
		return err
	}
	opts.Auth = trAuthMethod
	if cfg.Depth > 0 {
		opts.Depth = cfg.Depth
	}
	if cfg.Branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(cfg.Branch)
	}
	return nil
}

func configurePullOptions(cfg *RepositoryConfig, opts *git.PullOptions) error {
	opts.RemoteName = "origin"
	trAuthMethod, err := configureAuthOptions(cfg)
	if err != nil {
		return err
	}
	opts.Auth = trAuthMethod
	if cfg.Depth > 0 {
		opts.Depth = cfg.Depth
	}
	if cfg.Branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(cfg.Branch)
		opts.SingleBranch = true
	}
	return nil
}

func configureAuthOptions(cfg *RepositoryConfig) (transport.AuthMethod, error) {
	if cfg.Auth == nil {
		return nil, nil
	}
	cfg.Auth.KeyPath = expandDir(cfg.Auth.KeyPath)

	switch cfg.transport {
	case "http":
		// Configure authentication for HTTP/S.
		switch {
		case cfg.Auth.Username != "":
			return &http.BasicAuth{
				Username: cfg.Auth.Username,
				Password: cfg.Auth.Password,
			}, nil
		}
	case "ssh":
		// Configure authentication for SSH.
		switch {
		case cfg.Auth.KeyPath != "":
			var publicKeysUser string
			switch {
			case strings.Contains(cfg.Address, "@"):
				cfgAddressArr := strings.SplitN(cfg.Address, "@", 2)
				publicKeysUser = cfgAddressArr[0]
			case cfg.Auth.Username != "":
				publicKeysUser = cfg.Auth.Username
			}

			if publicKeysUser == "" {
				publicKeysUser = "git"
			}

			publicKeys, err := ssh.NewPublicKeysFromFile(publicKeysUser, cfg.Auth.KeyPath, cfg.Auth.KeyPassphrase)
			if err != nil {
				return nil, err
			}
			if cfg.Auth.StrictHostKeyCheckingDisabled {
				publicKeys.HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
					HostKeyCallback: cryptossh.InsecureIgnoreHostKey(),
				}
			}
			return publicKeys, nil
		case cfg.Auth.Username != "":
			password := &ssh.Password{
				User:     cfg.Auth.Username,
				Password: cfg.Auth.Password,
			}
			if cfg.Auth.StrictHostKeyCheckingDisabled {
				password.HostKeyCallbackHelper = ssh.HostKeyCallbackHelper{
					HostKeyCallback: cryptossh.InsecureIgnoreHostKey(),
				}
			}
			return password, nil
		}
	}
	return nil, nil
}

func expandDir(s string) string {
	if s == "" || !strings.HasPrefix(s, "~") {
		return s
	}
	hd, err := os.UserHomeDir()
	if err != nil {
		return s
	}
	output := hd + s[1:]
	return output
}

func autoUpdater(r *Repository) {
	r.logger.Debug(
		"auto-update enabled",
		zap.String("repo_name", r.Config.Name),
		zap.Int("interval", r.Config.UpdateInterval),
	)
	intervals := time.NewTicker(time.Second * time.Duration(r.Config.UpdateInterval))
	defer intervals.Stop()
	for range intervals.C {
		if r == nil {
			break
		}
		if err := r.update(); err != nil {
			r.logger.Error("failed auto-updating repo", zap.String("repo_name", r.Config.Name), zap.Error(err))
			continue
		}
		r.logger.Debug("auto-updated repo", zap.String("repo_name", r.Config.Name))
	}
	return
}
