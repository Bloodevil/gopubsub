package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

var addr = flag.String("addr", ":8080", "http service address")

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Client struct {
	// The websocket connection.
	conn *websocket.Conn
	send chan []byte
}

func main() {
    s := make(chan os.Signal, 1)
    signal.Notify(s,
        syscall.SIGHUP,
        syscall.SIGINT,
        syscall.SIGTERM,
        syscall.SIGQUIT,
        syscall.SIGKILL)

    upgrader.CheckOrigin = func(r *http.Request) bool {
            origin := r.Header["Origin"]
            if len(origin) == 0 {
                    return true
            }
            u, err := url.Parse(origin[0])
            if err != nil {
                    return false
            }
            log.Info("check origin", log.Params{"rOrigin": r.Host,
                    "uOrigin": u.Host,
                    "check":   util.EqualASCIIFold(u.Host, r.Host)})
            return true
    }

	http.HandleFunc("/channel", func(w http.ResponseWriter, r *http.Request) {
		ConnectWebSocketHandler(w, r)
	})

    go func() {
	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Println("Running error")
		fmt.Println(err)
	}
    }()

    sig := <-s

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := server.Shutdown(ctx); err != nil {
            log.Err("HTTP Server shutdown error", log.Params{"err": err})
    }
}

// api
func ConnectWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Fail to upgrade %s", err)
		return
	}
	fmt.Println("success to get websocket")

	client := &Client{
		conn:    conn,
		send:    make(chan []byte),
	}

	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.manager.broadcast <- message
	}
}

