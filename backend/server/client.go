package server

import (
	"context"
	"log"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	// clientSendBuffer bounds how far a client may fall behind before it is dropped.
	clientSendBuffer = 64
	writeTimeout     = 10 * time.Second
	// pingInterval keeps idle sockets alive through proxies and NATs, and
	// surfaces dead peers within pingInterval+writeTimeout.
	pingInterval = 30 * time.Second
)

// client is one websocket connection. Only writePump writes to conn.
type client struct {
	conn *websocket.Conn
	send chan *Message
}

func newClient(conn *websocket.Conn) *client {
	return &client{
		conn: conn,
		send: make(chan *Message, clientSendBuffer),
	}
}

// writePump drains the send queue and pings the peer until ctx is done, the
// hub drops the client, or a write fails. Any failure closes the connection,
// which ends the read loop and the client reconnects.
func (c *client) writePump(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-c.send:
			if !ok {
				if ctx.Err() == nil {
					_ = c.conn.Close(websocket.StatusPolicyViolation, "client too slow")
				}

				return
			}

			wctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := wsjson.Write(wctx, c.conn, msg)

			cancel()

			if err != nil {
				log.Printf("Error writing to websocket client: %v", err)

				_ = c.conn.CloseNow()

				return
			}

		case <-ticker.C:
			pctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pctx)

			cancel()

			if err != nil {
				log.Printf("Websocket ping failed, closing: %v", err)

				_ = c.conn.CloseNow()

				return
			}
		}
	}
}
