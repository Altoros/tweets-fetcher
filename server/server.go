package server

import (
	"net/http"

	log "github.com/inconshreveable/log15"

	"github.com/Altoros/tweets-fetcher/fetcher"
	"github.com/Altoros/tweets-fetcher/server/handlers"
)

type server struct {
	logger  log.Logger
	fetcher fetcher.Fetcher
	fanout  handlers.Fanout
}

type Server interface {
	Start(chan error, string)
	Stop()
}

func New(logger log.Logger, fetcher fetcher.Fetcher) Server {
	fanout := handlers.NewFanout(fetcher.Tweets())
	fanout.Run()
	return &server{
		logger:  logger.New("module", "server"),
		fetcher: fetcher,
		fanout:  fanout,
	}
}

func (s *server) Start(errCh chan error, port string) {
	s.logger.Info("Starting server", "port", port)
	mux := handlers.New(s.logger, s.fetcher, s.fanout, "templates")
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		errCh <- err
	}
}

func (s *server) Stop() {
	s.logger.Info("Stopping server")
	s.fanout.UnregisterAll()
}
