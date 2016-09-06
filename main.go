package main

import (
	"fmt"
	"os"
	"os/signal"

	log "github.com/inconshreveable/log15"

	"github.com/Altoros/tweets-fetcher/fetcher"
	"github.com/Altoros/tweets-fetcher/server"
)

var (
	defaultPort          = "8080"
	requiredEnvVariables = []string{"TWITTER_CONSUMER_KEY", "TWITTER_CONSUMER_SECRET",
		"TWITTER_CONSUMER_ACCESS_TOKEN", "TWITTER_CONSUMER_ACCESS_SECRET"}
)

func main() {
	logger := log.New("module", "main")
	logger.SetHandler(log.StreamHandler(os.Stdout, log.JsonFormat()))

	err := checkReqiredEnvVariables()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	fetcher := fetcher.New(logger, fetcher.Options{
		TwitterConsumerKey:    os.Getenv("TWITTER_CONSUMER_KEY"),
		TwitterConsumerSecret: os.Getenv("TWITTER_CONSUMER_SECRET"),
		TwitterAccessToken:    os.Getenv("TWITTER_CONSUMER_ACCESS_TOKEN"),
		TwitterAccessSecret:   os.Getenv("TWITTER_CONSUMER_ACCESS_SECRET"),
	})

	server := server.New(logger, fetcher)
	errChan := make(chan error)
	go server.Start(errChan, getPort())

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	select {
	case err := <-errChan:
		logger.Error("Start server", "err", err)
	case <-signalChan:
		logger.Info("SIGINT caught, exiting")
		server.Stop()
	}
}

func getPort() string {
	if os.Getenv("PORT") == "" {
		return defaultPort
	}
	return os.Getenv("PORT")
}

func checkReqiredEnvVariables() error {
	for _, variable := range requiredEnvVariables {
		if os.Getenv(variable) == "" {
			return fmt.Errorf("Env variable %s required, but is not set", variable)
		}
	}

	return nil
}
