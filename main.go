package main

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/clients/metrics"
	"github.com/alienrobotwizard/flotilla-os/core/app"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines/kubernetes"
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
	// Extra engines allows for creating and adding new execution engine types
	//
	var extraEngines []engines.Engine

	if c.IsSet(kubernetes.EngineName) {
		k8sEngine, err := kubernetes.NewEngine(c)
		if err != nil {
			fmt.Printf("%+v\n", errors.Wrap(err, "unable to initialize kubernetes engine"))
			os.Exit(1)
		}
		extraEngines = append(extraEngines, k8sEngine)
	}

	ctx := context.Background()
	server, err := app.NewApp(ctx, c, extraEngines...)
	if err != nil {
		fmt.Printf("%+v\n", errors.Wrap(err, "unable to initialize app server"))
		os.Exit(1)
	}

	log.Fatal(server.Run())
}
