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
	manager *ClientManager

	// The websocket connection.
	conn *websocket.Conn
	send chan []byte
}

func (cm *ClientManager) run() {
	for {
		select {
		case client := <-cm.register:
			cm.clients[client] = true
		case client := <-cm.unregister:
			if _, ok := cm.clients[client]; ok {
				delete(cm.clients, client)
				close(client.send)
			}
		case message := <-cm.broadcast:
			for client := range cm.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(cm.clients, client)
				}
			}
		}
	}
}

func main() {
	s := make(chan os.Signal, 1)
	signal.Notify(s,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL)

	cm := newClientManager()
	go cm.run()

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	http.HandleFunc("/channel", func(w http.ResponseWriter, r *http.Request) {
		ConnectWebSocketHandler(cm, w, r)
	})

	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Println("Running error")
		fmt.Println(err)
	}

}

// api
func ConnectWebSocketHandler(ClientManager *ClientManager, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Fail to upgrade %s", err)
		return
	}
	fmt.Println("success to get websocket")

	client := &Client{
		conn:    conn,
		send:    make(chan []byte),
		manager: ClientManager,
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

