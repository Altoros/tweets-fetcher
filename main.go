package main

import (
	//"fmt"
	"os"
	"os/signal"

	log "github.com/inconshreveable/log15"

	"github.com/Altoros/tweets-fetcher/fetcher"
	"github.com/Altoros/tweets-fetcher/server"
)

func main() {
	logger := log.New("module", "main")
	logger.SetHandler(log.StreamHandler(os.Stdout, log.JsonFormat()))

	fetcher := fetcher.New(logger)

	server := server.New(logger, fetcher)
	errChan := make(chan error)
	go server.Start(errChan)

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
