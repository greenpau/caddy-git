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
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/greenpau/caddy-git/pkg/service"
	"go.uber.org/zap"
)

var (
	appName = "git"

	// Interface guards
	_ caddy.Provisioner = (*App)(nil)
	_ caddy.Module      = (*App)(nil)
	_ caddy.App         = (*App)(nil)
)

func init() {
	caddy.RegisterModule(App{})
}

// App implements git repository manager.
type App struct {
	Name    string          `json:"-"`
	Config  *service.Config `json:"config,omitempty"`
	manager *service.Manager
	logger  *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (App) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  caddy.ModuleID(appName),
		New: func() caddy.Module { return new(App) },
	}
}

// Provision sets up the repo manager.
func (app *App) Provision(ctx caddy.Context) error {
	app.Name = appName
	app.logger = ctx.Logger(app)

	app.logger.Info(
		"provisioning app instance",
		zap.String("app", app.Name),
	)

	manager, err := service.NewManager(app.Config, app.logger)
	if err != nil {
		app.logger.Error(
			"failed configuring app instance",
			zap.String("app", app.Name),
			zap.Error(err),
		)
		return err
	}
	app.manager = manager

	app.logger.Info(
		"provisioned app instance",
		zap.String("app", app.Name),
	)
	return nil
}

// Start starts the App.
func (app App) Start() error {
	app.logger.Debug(
		"starting git repo manager",
		zap.String("app", app.Name),
	)

	if msgs := app.manager.Start(); msgs != nil {
		for _, msg := range msgs {
			app.logger.Error(
				"failed managing git repo",
				zap.String("app", app.Name),
				zap.String("repo", msg.Repository),
				zap.Error(msg.Error),
			)
		}
		return fmt.Errorf("git repo manager failed to start")
	}

	app.logger.Debug(
		"started git repo manager",
		zap.String("app", app.Name),
	)

	return nil
}

// Stop stops the App.
func (app App) Stop() error {
	app.logger.Debug(
		"stopping git repo manager",
		zap.String("app", app.Name),
	)

	if msgs := app.manager.Stop(); msgs != nil {
		for _, msg := range msgs {
			app.logger.Error(
				"failed stoppint git repo manager",
				zap.String("app", app.Name),
				zap.String("repo", msg.Repository),
				zap.Error(msg.Error),
			)
		}
		return fmt.Errorf("git repo manager failed to stop properly")
	}

	app.logger.Debug(
		"stopped git repo manager",
		zap.String("app", app.Name),
	)
	return nil
}
