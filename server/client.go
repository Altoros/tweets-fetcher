package server

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Altoros/tweets-fetcher/fetcher"
)

const (
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	connection       *websocket.Conn
	send             chan *fetcher.Tweet
	err              chan error
	done             chan bool
	handledSendClose chan bool
}

func (c *Client) write(mt int, payload []byte) error {
	// c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.connection.WriteMessage(mt, payload)
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.connection.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				err := c.write(websocket.CloseMessage, []byte{})
				if err != nil {
					c.err <- err
				} else {
					c.done <- true
				}
				c.handledSendClose <- true
				return
			}

			w, err := c.connection.NextWriter(websocket.TextMessage)
			if err != nil {
				c.err <- err
				return
			}

			js, err := json.Marshal(message)
			if err != nil {
				c.err <- err
			}
			w.Write([]byte(js))

			if err := w.Close(); err != nil {
				c.err <- err
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				c.err <- err
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.connection.Close()
	}()

	c.connection.SetReadDeadline(time.Now().Add(pongWait))
	c.connection.SetPongHandler(func(string) error { c.connection.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, _, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.err <- err
				return
			}
			break
		}
	}
}

func (c *Client) close() {
}
