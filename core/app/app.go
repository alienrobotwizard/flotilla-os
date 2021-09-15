package app

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/app/services"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
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

func NewApp(c *config.Config, sm state.Manager, engine engines.Engine) (App, error) {
	var (
		app App
		es  services.ExecutionService
		ts  services.TemplateService
		wm  *workers.WorkerManager
		err error
	)

	app.appContext, app.appShutdown = context.WithCancel(context.Background())

	if err = sm.Initialize(c); err != nil {
		return app, err
	}

	if err = engine.Initialize(app.appContext, c, sm); err != nil {
		return app, err
	}

	app.configure(c)

	if es, err = services.NewExecutionService(c, sm, engine); err != nil {
		return app, errors.Wrap(err, "problem initializing execution service")
	}

	if ts, err = services.NewTemplateService(c, sm); err != nil {
		return app, errors.Wrap(err, "problem initializing template service")
	}

	app.handler = Initialize(ts, es)
	if wm, err = workers.NewManager(c, sm, map[string]engines.Engine{"local": engine}); err != nil {
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

	wg, err := app.workerManager.Start(app.appContext)
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-signals:
			app.appShutdown()
			wg.Wait()
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
