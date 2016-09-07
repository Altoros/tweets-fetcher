package handlers

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/gorilla/websocket"
	log "github.com/inconshreveable/log15"

	"github.com/Altoros/tweets-fetcher/fetcher"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	homeTemplate *template.Template
)

func New(logger log.Logger, fetcher fetcher.Fetcher, fanout Fanout, templatesPath string) http.Handler {
	var err error

	mux := http.NewServeMux()
	homeTemplate, err = template.New("home").Delims("{{{", "}}}").ParseFiles(filepath.Join(templatesPath, "home.html"))
	if err != nil {
		panic(err)
	}
	handler := &fetcherHandler{
		logger:  logger,
		fetcher: fetcher,
		fanout:  fanout,
	}
	AttachRoutes(mux, handler)
	return mux
}

func AttachRoutes(mux *http.ServeMux, handler *fetcherHandler) {
	mux.HandleFunc("/", handler.home)
	mux.HandleFunc("/query", handler.query)
	mux.HandleFunc("/fetch", handler.fetch)
	mux.HandleFunc("/stop", handler.stop)
	mux.HandleFunc("/tweets", handler.tweets)
	staticHandler := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticHandler))
}

type fetcherHandler struct {
	logger  log.Logger
	fetcher fetcher.Fetcher
	fanout  Fanout
}

func (h *fetcherHandler) home(w http.ResponseWriter, r *http.Request) {
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
		h.logger.Error("Error rendering home page", "err", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}

func (h *fetcherHandler) query(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(h.fetcher.CurrentQuery()))
}

func (h *fetcherHandler) fetch(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "Request body is empty", http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Error reading request body", "err", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
	defer r.Body.Close()

	query := string(body)
	if query == "" {
		http.Error(w, "Query can't be blank", http.StatusBadRequest)
	}

	h.fetcher.Fetch(query)
}

func (h *fetcherHandler) stop(w http.ResponseWriter, r *http.Request) {
	h.fetcher.Stop()
	h.fanout.UnregisterAll()
}

func (h *fetcherHandler) tweets(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Error upgrading websocket", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.logger.Info("New client connected")

	client := &Client{
		connection:       connection,
		send:             make(chan *fetcher.Tweet, 256),
		err:              make(chan error),
		done:             make(chan bool),
		handledSendClose: make(chan bool),
	}
	h.fanout.Register(client)
	defer h.fanout.Unregister(client)

	go client.writePump()
	go client.readPump()

	select {
	case err := <-client.err:
		h.logger.Error("Socket error", "err", err)
		return
	case <-client.done:
		h.logger.Info("Client disconnected")
		return
	}
}
