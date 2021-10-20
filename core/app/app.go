package app

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/app/services"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines/local"
	"github.com/alienrobotwizard/flotilla-os/core/execution/workers"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	address            string
	mode               string
	corsAllowedOrigins []string
	logger             *log.Logger
	readTimeout        time.Duration
	writeTimeout       time.Duration
	handler            http.Handler
	workerManager      *workers.WorkerManager
	appContext         context.Context
	appShutdown        context.CancelFunc
}

func NewApp(ctx context.Context, c *config.Config, extraEngines ...engines.Engine) (App, error) {
	var (
		app App
		es  services.ExecutionService
		ts  services.TemplateService
		ws  services.WorkerService
		wm  *workers.WorkerManager
		err error
	)

	app.appContext, app.appShutdown = context.WithCancel(ctx)
	sm, err := state.NewSQLManager(app.appContext, c)
	if err != nil {
		return app, err
	}

	engine, err := local.NewLocalEngine(c)
	if err != nil {
		return app, err
	}

	app.configure(c)

	engineMap := map[string]engines.Engine{
		engine.Name(): engine,
	}

	for _, otherEngine := range extraEngines {
		engineMap[otherEngine.Name()] = otherEngine
	}

	//
	// Ensure all engines can actually clean up properly
	//
	go func(ctx context.Context, em map[string]engines.Engine) {
		select {
		case <-ctx.Done():
			for _, eng := range em {
				eng.Close()
			}
		}
	}(app.appContext, engineMap)

	if es, err = services.NewExecutionService(c, sm, engineMap); err != nil {
		return app, errors.Wrap(err, "problem initializing execution service")
	}

	if ts, err = services.NewTemplateService(c, sm); err != nil {
		return app, errors.Wrap(err, "problem initializing template service")
	}

	if ws, err = services.NewWorkerService(c, sm); err != nil {
		return app, errors.Wrap(err, "problem initializing worker service")
	}

	app.handler = Initialize(ts, es, ws)
	if wm, err = workers.NewManager(c, sm, engineMap); err != nil {
		return app, errors.Wrap(err, "problem initializing worker manager")
	}

	app.workerManager = wm
	return app, nil
}

func (app *App) Run() error {
	srv := &http.Server{
		Addr:         app.address,
		Handler:      app.handler,
		ReadTimeout:  app.readTimeout,
		WriteTimeout: app.writeTimeout,
	}

	// Set up worker manager and properly listen to signals for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	defer func() {
		signal.Stop(signals)
		app.appShutdown()
	}()

	app.workerManager.Start(app.appContext)

	go func() {
		select {
		case <-signals:
			app.appShutdown()
			srv.Shutdown(app.appContext)
		case <-app.appContext.Done():
			fmt.Println("context is done")
		}
	}()
	//

	return srv.ListenAndServe()
}

func (app *App) configure(c *config.Config) {
	app.address = ":5000"

	app.readTimeout = time.Duration(5) * time.Second
	app.writeTimeout = time.Duration(10) * time.Second

	if c.IsSet("http.server.listen_address") {
		app.address = c.GetString("http.server.listen_address")
	}

	if c.IsSet("http.server.read_timeout_seconds") {
		app.readTimeout = time.Duration(c.GetInt("http.server.read_timeout_seconds")) * time.Second
	}

	if c.IsSet("http.server.write_timeout_seconds") {
		app.writeTimeout = time.Duration(c.GetInt("http.server.write_timeout_seconds")) * time.Second
	}

	app.mode = c.GetString("flotilla_mode")
	app.corsAllowedOrigins = c.GetStringSlice("http.server.cors_allowed_origins")
}
