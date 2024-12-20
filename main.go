package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var debugFlag bool
var jsonLog bool
var port int
var configPath string
var cycleInterval int

var prefixes []*Prefix

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	flag.IntVar(&port, "port", 2113, "port")
	flag.BoolVar(&debugFlag, "debug", false, "debug")
	flag.BoolVar(&jsonLog, "json", false, "json logging")
	flag.StringVar(&configPath, "config", "config.toml", "config file")
	flag.IntVar(&cycleInterval, "i", 10, "cycle interval")
}

func main() {
	flag.Parse()

	if !jsonLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	if debugFlag {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug log enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if err := loadConfig(); err != nil {
		log.Fatal().Err(err).Msg("error loading config")
		return
	}

	bgpInit()

	log.Info().
		Msg("Starting PEERINGMON Controller")

	prefixes = prefixesInit()

	http.Handle("/metrics", promhttp.Handler())

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr: ":" + strconv.Itoa(port),
	}

	go func() {
		cycle()

		ticker := time.NewTicker(time.Duration(cycleInterval) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			cycle()
		}
	}()

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()
	log.Info().
		Int("port", port).
		Int("cycle_interval", cycleInterval).
		Msg("Started controller")

	go startMemoryCleanup()

	<-done
	log.Info().Msg("Stopping")
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := bgpStop(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Failed to gracefully stop bgp instance")
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Failed to gracefully stop http server")
	}
	log.Info().Msg("Graceful Shutdown Successful, bye")
}

func startMemoryCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		debug.FreeOSMemory()
	}
}
