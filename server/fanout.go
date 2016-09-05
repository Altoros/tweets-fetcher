package server

import (
	"github.com/Altoros/tweets-fetcher/fetcher"
)

type fanout struct {
	input   chan *fetcher.Tweet
	clients map[*Client]bool
}

func (f *fanout) Register(client *Client) {
	f.clients[client] = true
}

func (f *fanout) Unregister(client *Client) {
	if _, ok := f.clients[client]; ok {
		close(client.send)
		delete(f.clients, client)
		<-client.handledSendClose
	}
}

func (f *fanout) Run() {
	go func() {
		for msg := range f.input {
			for client, _ := range f.clients {
				client.send <- msg
			}
		}
	}()
}

func (f *fanout) UnregisterAll() {
	for client, _ := range f.clients {
		f.Unregister(client)
	}
}
