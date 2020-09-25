package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsMessage struct {
	MessageType string      `json:"type"`
	Payload     interface{} `json:"payload"`
}

type wsClient struct {
	pool *wsPool
	conn *websocket.Conn
	send chan *wsMessage
}

func (c *wsClient) readPump() {
	defer func() {
		c.pool.unregister <- c
		c.conn.Close()
	}()
	for {
		var m wsMessage

		err := c.conn.ReadJSON(&m)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		log.Printf("error: %v", m)

		err = c.conn.WriteJSON(m)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func (c *wsClient) writePump() {
	defer c.conn.Close()
	for {
		select {
		case msg := <-c.send:
			log.Println(msg)
			if err := c.conn.WriteJSON(msg); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

type wsPool struct {
	// Registered clients.
	clients map[*wsClient]bool
	// Inbound messages from the clients.
	broadcast chan *wsMessage
	// Register requests from the clients.
	register chan *wsClient
	// Unregister requests from clients.
	unregister chan *wsClient
}

func newPool() *wsPool {
	return &wsPool{
		broadcast:  make(chan *wsMessage),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		clients:    make(map[*wsClient]bool),
	}
}

func (p *wsPool) start() {
	for {
		select {
		case client := <-p.register:
			p.clients[client] = true
		case client := <-p.unregister:
			if _, ok := p.clients[client]; ok {
				delete(p.clients, client)
				close(client.send)
			}
		case m := <-p.broadcast:
			for client := range p.clients {
				select {
				case client.send <- m:
				default:
					delete(p.clients, client)
					close(client.send)
				}
			}
		}
	}
}

func serveWs(pool *wsPool, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &wsClient{pool: pool, conn: conn, send: make(chan *wsMessage)}
	client.pool.register <- client

	go client.readPump()
	go client.writePump()

	payload := getAccumulatedRecords()
	client.send <- &wsMessage{"DEVICE_DATA_ALL", payload}
}
