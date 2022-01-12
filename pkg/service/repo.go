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
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"go.uber.org/zap"
	"os"
	"path"
	"sync"
	"time"
)

// Repository is a configuration for a command or app.
type Repository struct {
	Config      *RepositoryConfig `json:"config,omitempty"`
	mu          sync.Mutex
	logger      *zap.Logger
	lastUpdated time.Time
}

// NewRepository returns an instance of Repository.
func NewRepository(rc *RepositoryConfig) (*Repository, error) {
	r := &Repository{
		Config: rc,
	}
	return r, nil
}

func (r *Repository) update() error {
	r.mu.Lock()
	defer r.mu.Unlock()

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
		opts := &git.CloneOptions{
			URL: r.Config.Address,
		}
		if r.Config.Depth > 0 {
			opts.Depth = r.Config.Depth
		}
		if r.Config.Branch != "" {
			opts.ReferenceName = plumbing.NewBranchReferenceName(r.Config.Branch)
		}
		if _, err := git.PlainClone(repoDir, false, opts); err != nil {
			return err
		}
		return nil
	}
	// Pull the repository.

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
