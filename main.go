package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var debug bool
var jsonLog bool
var port int
var configPath string
var cycleInterval int

var prefixes []*Prefix

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	flag.IntVar(&port, "port", 2113, "port")
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.BoolVar(&jsonLog, "json", false, "json logging")
	flag.StringVar(&configPath, "config", "config.toml", "config file")
	flag.IntVar(&cycleInterval, "i", 10, "cycle interval")
}

func main() {
	flag.Parse()

	if !jsonLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	if debug {
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
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()
	log.Info().
		Int("port", port).
		Int("cycle_interval", cycleInterval).
		Msg("Started controller")

	cycle()
	go func() {
		ticker := time.NewTicker(time.Duration(cycleInterval) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			cycle()
		}
	}()

	go func() {
		//debug purposes
		ticker := time.NewTicker(time.Duration(30) * time.Second)
		defer ticker.Stop()

		v4Family := &api.Family{
			Afi:  api.Family_AFI_IP,
			Safi: api.Family_SAFI_UNICAST,
		}
		for range ticker.C {
			fmt.Println("called")
			s.ListPath(context.Background(), &api.ListPathRequest{Family: v4Family}, func(p *api.Destination) {
				fmt.Println(p)
			})
		}
	}()

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
