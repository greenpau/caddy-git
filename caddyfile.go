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
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/greenpau/caddy-git/pkg/service"
	"strconv"
	"strings"
)

func init() {
	httpcaddyfile.RegisterGlobalOption("git", parseCaddyfileAppConfig)
	httpcaddyfile.RegisterDirective("git", getRouteFromParseCaddyfileHandlerConfig)
}

// parseCaddyfileAppConfig sets up a repo manager.
//
// Syntax:
//
// git {
//   repo <name> {
//     base_dir <path>
//     url <path>
//     auth key <path> [passphrase <passphrase>] [no_strict_host_key_check]
//     auth username <username> password <password>
//     webhook <name> <header> <secret>
//     branch <name>
//     depth 1
//     update every <seconds>
//   }

// parseCaddyfileHandlerConfig configures repo update handler.
//
// Syntax:
//
// route /update {
//   git update repo <name>
// }

const badRepl string = "ERROR_BAD_REPL"

var argRules = map[string]argRule{
	"base_dir": argRule{Min: 1, Max: 1},
	"url":      argRule{Min: 1, Max: 1},
	"auth":     argRule{Min: 2, Max: 255},
	"branch":   argRule{Min: 1, Max: 1},
	"depth":    argRule{Min: 1, Max: 1},
	"update":   argRule{Min: 1, Max: 255},
	"webhook":  argRule{Min: 3, Max: 3},
	"post":     argRule{Min: 2, Max: 2},
}

type argRule struct {
	Min int
	Max int
}

func parseCaddyfileAppConfig(d *caddyfile.Dispenser, _ interface{}) (interface{}, error) {
	repl := caddy.NewReplacer()
	app := new(App)
	app.Config = service.NewConfig()

	if !d.Next() {
		return nil, d.ArgErr()
	}

	for d.NextBlock(0) {
		switch d.Val() {
		case "repo":
			args := d.RemainingArgs()
			if len(args) != 1 {
				return nil, d.ArgErr()
			}
			rc := service.NewRepositoryConfig()
			rc.Name = args[0]
			for nesting := d.Nesting(); d.NextBlock(nesting); {
				k := d.Val()
				v := findReplace(repl, d.RemainingArgs())
				if _, exists := argRules[k]; exists {
					if err := validateArg(k, v); err != nil {
						return nil, d.Errf("%s", err)
					}
				}
				switch k {
				case "base_dir":
					rc.BaseDir = v[0]
				case "url":
					rc.Address = v[0]
				case "auth":
					authCfg := &service.AuthConfig{}
					switch v[0] {
					case "key":
						if len(v) < 2 {
							return nil, d.Errf("malformed %q directive: %v", k, v)
						}
						authCfg.KeyPath = v[1]
						if len(v) > 2 {
							if v[2] == "passphrase" {
								authCfg.KeyPassphrase = v[3]
							}
						}
					case "username":
						if len(v) < 4 {
							return nil, d.Errf("malformed %q directive", k)
						}
						authCfg.Username = v[1]
						if v[2] == "password" {
							authCfg.Password = v[3]
						}
					}
					if findString(v, "no_strict_host_key_check") {
						authCfg.StrictHostKeyCheckingDisabled = true
					}
					rc.Auth = authCfg
				case "webhook":
					whCfg := &service.WebhookConfig{
						Name:   v[0],
						Header: v[1],
						Secret: v[2],
					}
					rc.Webhooks = append(rc.Webhooks, whCfg)
				case "branch":
					rc.Branch = v[0]
				case "depth":
					if n, err := strconv.Atoi(v[0]); err == nil {
						rc.Depth = n
					} else {
						return nil, d.Errf("%s value %q is not integer", k, v[0])
					}
					// return nil, d.Errf("the depth directive is disabled due to the issue with github.com/go-git/go-git")
				case "post":
					switch {
					case strings.Join(v, " ") == "pull exec":
						ppeCfg := &service.ExecConfig{}
						for nesting := d.Nesting(); d.NextBlock(nesting); {
							nk := d.Val()
							nargs := findReplace(repl, d.RemainingArgs())
							switch nk {
							case "name":
								ppeCfg.Name = nargs[0]
							case "command":
								ppeCfg.Command = nargs[0]
							case "args":
								ppeCfg.Args = nargs
							default:
								return nil, d.Errf("malformed %q directive: %v", nk, nargs)
							}
						}
						rc.PostPullExec = append(rc.PostPullExec, ppeCfg)
					default:
						return nil, d.Errf("malformed %q directive: %v", k, v)
					}
				case "update":
					if len(v) != 2 {
						return nil, d.Errf("malformed %q directive: %v", k, v)
					}
					if v[0] != "every" {
						return nil, d.Errf("malformed %q directive: %v", k, v)
					}
					if n, err := strconv.Atoi(v[1]); err == nil {
						rc.UpdateInterval = n
					} else {
						return nil, d.Errf("%s value %q is not integer", k, v[0])
					}
				default:
					return nil, d.Errf("unsupported %q key", k)
				}
			}
			if err := app.Config.AddRepository(rc); err != nil {
				return nil, d.Err(err.Error())
			}
		default:
			return nil, d.ArgErr()
		}
	}

	return httpcaddyfile.App{
		Name:  appName,
		Value: caddyconfig.JSON(app, nil),
	}, nil
}

func validateArg(k string, v []string) error {
	r, exists := argRules[k]
	if !exists {
		return nil
	}
	if r.Min > len(v) {
		return fmt.Errorf("too few args for %q directive (config: %d, min: %d)", k, len(v), r.Min)
	}
	if r.Max < len(v) {
		return fmt.Errorf("too many args for %q directive (config: %d, max: %d", k, len(v), r.Max)
	}
	return nil
}

func parseCaddyfileHandlerConfig(h httpcaddyfile.Helper) (*service.Endpoint, error) {
	endpoint := &service.Endpoint{}

	for h.Next() {
		args := h.RemainingArgs()
		strArgs := strings.Join(args, " ")
		if !strings.Contains(strArgs, "update repo ") {
			return nil, h.Errf("unsupported config: git %s", strArgs)
		}
		switch {
		case args[0] == "update" && args[1] == "repo":
			if len(args) != 3 {
				return nil, h.Errf("malformed config: git %s", strArgs)
			}
			endpoint.Path = "*"
			endpoint.RepositoryName = args[2]
		case args[1] == "update" && args[2] == "repo":
			if len(args) != 4 {
				return nil, h.Errf("malformed config: git %s", strArgs)
			}
			endpoint.Path = args[0]
			endpoint.RepositoryName = args[3]
		default:
			return nil, h.Errf("malformed config: git %s", strArgs)
		}
	}

	h.Reset()
	h.Next()
	return endpoint, nil
}

func getRouteFromParseCaddyfileHandlerConfig(h httpcaddyfile.Helper) ([]httpcaddyfile.ConfigValue, error) {
	endpoint, err := parseCaddyfileHandlerConfig(h)
	if err != nil {
		return nil, err
	}
	pathMatcher := caddy.ModuleMap{
		"path": h.JSON(caddyhttp.MatchPath{endpoint.Path}),
	}
	route := caddyhttp.Route{
		HandlersRaw: []json.RawMessage{
			caddyconfig.JSONModuleObject(&Middleware{Endpoint: endpoint}, "handler", "git", nil),
		},
	}
	subroute := new(caddyhttp.Subroute)
	subroute.Routes = append([]caddyhttp.Route{route}, subroute.Routes...)
	return h.NewRoute(pathMatcher, subroute), nil

}

func findReplace(repl *caddy.Replacer, arr []string) (output []string) {
	for _, item := range arr {
		output = append(output, repl.ReplaceAll(item, badRepl))
	}
	return output
}

func findString(arr []string, s string) bool {
	for _, x := range arr {
		if x == s {
			return true
		}
	}
	return false
}
