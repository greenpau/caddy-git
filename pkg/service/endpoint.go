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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Endpoint handles git management requests.
type Endpoint struct {
	mu             sync.Mutex
	Name           string `json:"-"`
	Path           string `json:"path,omitempty" xml:"path,omitempty" yaml:"path,omitempty"`
	RepositoryName string
	logger         *zap.Logger
	startedAt      time.Time
}

// SetLogger add logger to Endpoint.
func (m *Endpoint) SetLogger(logger *zap.Logger) {
	m.logger = logger
}

// Provision configures the instance of Endpoint.
func (m *Endpoint) Provision() error {
	m.startedAt = time.Now().UTC()
	m.Name = "git-" + m.RepositoryName

	m.logger.Info(
		"provisioned plugin instance",
		zap.String("instance_name", m.Name),
		zap.String("path", m.Path),
		zap.Time("started_at", m.startedAt),
	)
	return nil
}

// Validate implements caddy.Validator.
func (m *Endpoint) Validate() error {
	m.logger.Info(
		"validated plugin instance",
		zap.String("instance_name", m.Name),
	)
	return nil
}

// ServeHTTP serves git management requests.
func (m *Endpoint) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	m.logger.Debug(
		"received update git repo request",
		zap.String("repo_name", m.RepositoryName),
	)

	resp := make(map[string]interface{})
	repo, exists := manager.repos[m.RepositoryName]
	if !exists {
		resp["status_code"] = http.StatusInternalServerError
		m.logger.Warn("repo not found", zap.String("repo_name", m.RepositoryName))
		return m.respondHTTP(ctx, w, r, resp)
	}

	if len(repo.Config.Webhooks) > 0 {
		// Inspect HTTP headers for webhooks.
		var authorized bool
		for _, webhook := range repo.Config.Webhooks {
			hdr := r.Header.Get(webhook.Header)
			if hdr == "" {
				continue
			}

			var authFailed bool
			var authFailMessage string

			switch webhook.Header {
			case "X-Hub-Signature-256", strings.ToUpper("X-Hub-Signature-256"):
				if r.Method != "POST" {
					authFailed = true
					authFailMessage = "non-POST request"
					break
				}
				hdrParts := strings.SplitN(hdr, "=", 2)
				if len(hdrParts) != 2 {
					authFailed = true
					authFailMessage = fmt.Sprintf("malformed %s header", webhook.Header)
					break
				}
				if hdrParts[0] != "sha256" {
					authFailMessage = fmt.Sprintf("malformed %s header, sha256 not found", webhook.Header)
				}
				if err := validateSignature(r, strings.TrimSpace(hdrParts[1]), webhook.Secret); err != nil {
					authFailed = true
					authFailMessage = fmt.Sprintf("signature validation failed: %v", err)
				}
			default:
				if hdr != webhook.Secret {
					authFailed = true
					authFailMessage = "auth header value mismatch"
				}
			}

			if authFailed {
				resp["status_code"] = http.StatusUnauthorized
				m.logger.Warn(
					"webhook authentication failed",
					zap.String("repo_name", repo.Config.Name),
					zap.String("webhook_header", webhook.Header),
					zap.String("error", authFailMessage),
				)
				return m.respondHTTP(ctx, w, r, resp)
			}

			authorized = true
			break
		}

		if !authorized {
			resp["status_code"] = http.StatusUnauthorized
			m.logger.Warn(
				"webhook authentication failed",
				zap.String("repo_name", repo.Config.Name),
				zap.String("error", "auth header not found"),
			)
			return m.respondHTTP(ctx, w, r, resp)
		}
	}

	if err := repo.update(); err != nil {
		m.logger.Warn("failed updating repo", zap.String("repo_name", repo.Config.Name), zap.Error(err))
		resp["status_code"] = http.StatusInternalServerError
		return m.respondHTTP(ctx, w, r, resp)
	}

	resp["status_code"] = http.StatusOK
	return m.respondHTTP(ctx, w, r, resp)
}

func (m *Endpoint) respondHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, data map[string]interface{}) error {
	b, _ := json.Marshal(data)
	if code, exists := data["status_code"]; exists {
		w.WriteHeader(code.(int))
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write(b)
	return nil
}

func validateSignature(r *http.Request, wantSig, secret string) error {
	if wantSig == "" {
		return fmt.Errorf("empty signature")
	}
	if len(wantSig) != 64 {
		return fmt.Errorf("malformed sha256 hash, length %d", len(wantSig))
	}

	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed reading request body")
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(respBody)
	gotSig := hex.EncodeToString(h.Sum(nil))
	if wantSig != gotSig {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}
