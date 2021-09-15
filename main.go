package main

import (
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/clients/metrics"
	"github.com/alienrobotwizard/flotilla-os/core/app"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines/local"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/pkg/errors"
	"log"
	"os"
)

func main() {

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: flotilla-os <conf_dir>")
		os.Exit(1)
	}

	//
	// Wrap viper for configuration
	//
	confDir := args[1]
	c, err := config.NewConfig(&confDir)
	if err != nil {
		fmt.Printf("%+v\n", errors.Wrap(err, "unable to initialize config"))
		os.Exit(1)
	}

	//
	// Instantiate metrics client.
	//
	if err = metrics.InstantiateClient(c); err != nil {
		fmt.Printf("%+v\n", errors.Wrap(err, "unable to initialize metrics client"))
		os.Exit(1)
	}

	//
	// Get state manager for reading and writing
	// state about definitions and runs
	//
	var (
		engine       local.Engine
		stateManager state.SQLManager
	)

	server, err := app.NewApp(c, &stateManager, &engine)
	if err != nil {
		fmt.Printf("%+v\n", errors.Wrap(err, "unable to initialize app server"))
		os.Exit(1)
	}

	log.Fatal(server.Run())
}
