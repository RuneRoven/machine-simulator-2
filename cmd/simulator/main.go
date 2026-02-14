package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/app"
	"machine-simulator-2/internal/config"
	_ "machine-simulator-2/internal/machine/types"
)

var (
	profileFlag   = flag.String("profile", "", "Name of the profile to load (e.g., demo-mixed)")
	configDirFlag = flag.String("config-dir", "config", "Path to configuration directory")
)

func main() {
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	configDir := *configDirFlag

	// Try per-line env vars first (SIMULATOR_LINE_*)
	cfg, err := config.LoadFromLineEnvVars(configDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration from line env vars")
	}

	if cfg != nil {
		log.Info().Int("lines", len(cfg.Lines)).Msg("Configuration loaded from SIMULATOR_LINE_* env vars")
	} else {
		// Fall back to profile-based loading
		profile := *profileFlag
		if profile == "" {
			profile = os.Getenv("SIMULATOR_PROFILE")
		}
		if profile == "" {
			fmt.Fprintln(os.Stderr, "Error: no profile specified. Use --profile <name> or set SIMULATOR_PROFILE.")
			fmt.Fprintln(os.Stderr, "You can also set SIMULATOR_LINE_* env vars to select individual lines.")
			fmt.Fprintln(os.Stderr, "Available profiles are in config/profiles/")
			os.Exit(1)
		}

		log.Info().Str("profile", profile).Msg("Loading configuration from profile")

		cfg, err = config.Load(configDir, profile)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to load configuration")
		}
	}

	// Set log level
	switch cfg.Global.LogLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Info().
		Str("profile", cfg.Profile.Name).
		Int("lines", len(cfg.Lines)).
		Msg("Starting machine simulator")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("Shutdown signal received")
		cancel()
	}()

	application, err := app.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize application")
	}

	if err := application.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("Application error")
	}
}
