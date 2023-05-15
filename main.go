package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/DerGut/emojitransport/emoji"
)

const configPath = "config.json"

func main() {
	config, err := emoji.ParseConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to parse config: %s\n", err)
		os.Exit(1)
	}

	if err := run(config); err != nil {
		fmt.Printf("Failed to run transport: %s\n", err)
		os.Exit(1)
	}
}

func run(config emoji.Config) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	store, err := emoji.NewStore(config.Directory)
	if err != nil {
		return fmt.Errorf("crete store: %w", err)
	}
	defer store.Close()

	exporter := emoji.NewSlackExporter(store, config)

	if err := exporter.Run(ctx); err != nil {
		return fmt.Errorf("exporter run: %w", err)
	}

	return nil
}
