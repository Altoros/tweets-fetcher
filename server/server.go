package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"

	"github.com/gorilla/websocket"
	log "github.com/inconshreveable/log15"

	"github.com/Altoros/tweets-fetcher/fetcher"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	homeTemplate *template.Template
)

type server struct {
	logger  log.Logger
	fetcher fetcher.Fetcher
	fanout  *fanout
}

type Server interface {
	Start(chan error, string)
	Stop()
}

func New(logger log.Logger, fetcher fetcher.Fetcher) Server {
	fanout := fanout{
		input:   fetcher.Tweets(),
		clients: make(map[*Client]bool),
	}
	return &server{
		logger:  logger.New("module", "server"),
		fetcher: fetcher,
		fanout:  &fanout,
	}
}

func (s *server) Start(errCh chan error, port string) {
	s.logger.Info("Starting server", "port", port)

	var err error

	homeTemplate, err = template.New("home").Delims("{{{", "}}}").ParseFiles("home.html")
	if err != nil {
		errCh <- fmt.Errorf("Error parsing home template: %s", err)
	}

	s.fanout.Run()

	http.HandleFunc("/", s.homeHandler)
	http.HandleFunc("/query", s.queryHandler)
	http.HandleFunc("/fetch", s.fetchHandler)
	http.HandleFunc("/stop", s.stopHandler)
	http.HandleFunc("/tweets", s.tweetsWsHandler)
	staticHandler := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticHandler))

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		errCh <- err
	}
}

func (s *server) Stop() {
	s.logger.Info("Stopping server")
	s.fanout.UnregisterAll()
}

func (s *server) homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Home handler")
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := homeTemplate.ExecuteTemplate(w, "home.html", r.Host)
	if err != nil {
		s.logger.Error("Error rendering home page", "err", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}

func (s *server) queryHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(s.fetcher.CurrentQuery()))
}

func (s *server) fetchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "Request body is empty", http.StatusBadRequest)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Error reading request body", "err", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
	defer r.Body.Close()

	query := string(body)
	if query == "" {
		http.Error(w, "Query can't be blank", http.StatusBadRequest)
	}

	s.fetcher.Fetch(query)
}

func (s *server) stopHandler(w http.ResponseWriter, r *http.Request) {
	s.fetcher.Stop()
	s.fanout.UnregisterAll()
}

func (s *server) tweetsWsHandler(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Error upgrading websocket", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	s.logger.Info("New client connected")

	client := &Client{
		connection:       connection,
		send:             make(chan *fetcher.Tweet, 256),
		err:              make(chan error),
		done:             make(chan bool),
		handledSendClose: make(chan bool),
	}
	s.fanout.Register(client)
	defer s.fanout.Unregister(client)

	go client.writePump()
	go client.readPump()

	select {
	case err := <-client.err:
		s.logger.Error("Socket error", "err", err)
		return
	case <-client.done:
		s.logger.Info("Client disconnected")
		return
	}
}
