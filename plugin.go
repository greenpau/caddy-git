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
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/greenpau/caddy-git/pkg/service"
)

func init() {
	caddy.RegisterModule(Middleware{})
}

// Middleware implements git repository manager.
type Middleware struct {
	Endpoint *service.Endpoint `json:"endpoint,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (Middleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.git",
		New: func() caddy.Module { return new(Middleware) },
	}
}

// UnmarshalCaddyfile unmarshals a Caddyfile.
func (m *Middleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) (err error) {
	endpoint, err := parseCaddyfileHandlerConfig(httpcaddyfile.Helper{Dispenser: d})
	if err != nil {
		return err
	}
	m.Endpoint = endpoint
	return nil
}

// Provision provisions git repository endpoint.
func (m *Middleware) Provision(ctx caddy.Context) error {
	m.Endpoint.SetLogger(ctx.Logger(m))
	return m.Endpoint.Provision()
}

// Validate implements caddy.Validator.
func (m *Middleware) Validate() error {
	return m.Endpoint.Validate()
}

// ServeHTTP performs git repository management tasks.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, _ caddyhttp.Handler) error {
	return m.Endpoint.ServeHTTP(r.Context(), w, r)
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Middleware)(nil)
	_ caddy.Validator             = (*Middleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*Middleware)(nil)
	_ caddyfile.Unmarshaler       = (*Middleware)(nil)
)
