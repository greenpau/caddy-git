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
//     auth key <path> [passcode <passcode>
//     auth username <username> password <password>
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

var argRules = map[string]argRule{
	"base_dir": argRule{Min: 1, Max: 1},
	"url":      argRule{Min: 1, Max: 1},
	"auth":     argRule{Min: 2, Max: 5},
	"branch":   argRule{Min: 1, Max: 1},
	"depth":    argRule{Min: 1, Max: 1},
	"update":   argRule{Min: 1, Max: 255},
}

type argRule struct {
	Min int
	Max int
}

func parseCaddyfileAppConfig(d *caddyfile.Dispenser, _ interface{}) (interface{}, error) {
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
				v := d.RemainingArgs()
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
						switch len(v) {
						case 2:
							authCfg.KeyPath = v[1]
						case 4:
							authCfg.KeyPath = v[1]
							authCfg.KeyPassphrase = v[3]
						default:
							return nil, d.Errf("malformed %q directive", k)
						}
					case "username":
						if len(v) != 4 {
							return nil, d.Errf("malformed %q directive", k)
						}
						authCfg.Username = v[1]
						authCfg.Password = v[3]
					}
					rc.Auth = authCfg
				case "branch":
					rc.Branch = v[0]
				case "depth":
					if n, err := strconv.Atoi(v[0]); err == nil {
						rc.Depth = n
					} else {
						return nil, d.Errf("%s value %q is not integer", k, v[0])
					}
				case "update":
					return nil, d.Errf("unsupported %q key", k)
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
		return fmt.Errorf("too few args for %q directive", k)
	}
	if r.Max < len(v) {
		return fmt.Errorf("too many args for %q directive", k)
	}
	return nil
}

func parseCaddyfileHandlerConfig(h httpcaddyfile.Helper) (*service.Endpoint, error) {
	endpoint := &service.Endpoint{}

	for h.Next() {
		args := h.RemainingArgs()
		switch {
		case strings.HasPrefix(strings.Join(args, " "), "update repo "):
			endpoint.RepositoryName = args[2]
		default:
			return nil, h.Errf("unsupported config: git %s", strings.Join(args, " "))
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
		"path": h.JSON(caddyhttp.MatchPath{"*"}),
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
