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

package git

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/google/go-cmp/cmp"
)

const tf string = "Testfile"

func TestParseCaddyfileAppConfig(t *testing.T) {
	testcases := []struct {
		name      string
		d         *caddyfile.Dispenser
		want      string
		shouldErr bool
		err       error
	}{
		{
			name: "test parse config",
			d: caddyfile.NewTestDispenser(`
            git {
              repo authp.github.io {
                base_dir /tmp
                url https://github.com/authp/authp.github.io.git
                branch gh-pages
                depth 1
              }
            }`),
			want: `{
			  "config": {
                "repositories": [
                  {
                    "address":  "https://github.com/authp/authp.github.io.git",
                    "base_dir": "/tmp",
                    "branch":   "gh-pages",
                    "depth":    1,
                    "name":     "authp.github.io"
                  }
                ]
              }
			}`,
		},
		{
			name: "test parse config with unsupported bar key",
			d: caddyfile.NewTestDispenser(`
            git {
              repo bar {
                bar baz
              }
            }`),
			shouldErr: true,
			err:       fmt.Errorf("%s:%d - Error during parsing: unsupported %q key", tf, 4, "bar"),
		},
		{
			name: "test parse config with too few arg for repo arg",
			d: caddyfile.NewTestDispenser(`
            git {
              repo foo {
                url
              }
            }`),
			shouldErr: true,
			err:       fmt.Errorf("%s:%d - Error during parsing: too few args for %q directive", tf, 4, "url"),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			app, err := parseCaddyfileAppConfig(tc.d, nil)
			if err != nil {
				if !tc.shouldErr {
					t.Fatalf("expected success, got: %v", err)
				}
				if diff := cmp.Diff(err.Error(), tc.err.Error()); diff != "" {
					t.Fatalf("unexpected error: %v, want: %v", err, tc.err)
				}
				return
			}
			if tc.shouldErr {
				t.Fatalf("unexpected success, want: %v", tc.err)
			}
			got := unpack(t, string(app.(httpcaddyfile.App).Value))
			want := unpack(t, tc.want)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("parseCaddyfileAppConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func unpack(t *testing.T, s string) (m map[string]interface{}) {
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("failed to parse %q: %v", s, err)
	}
	return m
}
