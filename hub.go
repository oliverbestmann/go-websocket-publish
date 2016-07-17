package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
)

// number of packages to buffer for each client.
const SendChannelSize = 8

type Message struct {
	Type    int
	Payload []byte
}

type Broadcaster interface {
	Broadcast(message Message)

	BroadcastObject(message interface{}) error
}

type Hub struct {
	// write to this channel to publish a message
	broadcast chan Message

	// all the currently registered connections
	connections map[*hubConnection]bool

	// register requests
	register chan *hubConnection

	// unregister requests
	unregister chan *hubConnection
}

func NewHub() *Hub {
	return &Hub{
		broadcast:   make(chan Message, 16),
		register:    make(chan *hubConnection),
		unregister:  make(chan *hubConnection),
		connections: make(map[*hubConnection]bool),
	}
}

/**
 * Main loop for this hub. This method blocks forever, so it is best to
 * call it in a go-routine.
 */
func (h *Hub) MainLoop() {
	for {
		select {
		case conn := <-h.register:
			h.connections[conn] = true

		case conn := <-h.unregister:
			delete(h.connections, conn)
			close(conn.send)

		case message := <-h.broadcast:
			for conn := range h.connections {

				// send the message without blocking.
				// close the connection, if send queue is full.
				select {
				case conn.send <- message:
				default:
					delete(h.connections, conn)
					close(conn.send)
				}
			}
		}
	}
}

/**
 * Handles the given websocket connection. This call blocks
 * until the connection was closed.
 */
func (h *Hub) HandleConnection(socket *websocket.Conn) {

	// make a new connection object
	conn := &hubConnection{
		socket: socket,
		send:   make(chan Message, SendChannelSize),
	}

	// register this connection with the hub
	h.register <- conn

	// forward writes and consume the read end too.
	go conn.writeLoop()
	conn.readLoop()
}

func (h *Hub) Shutdown() {
	// TODO
}

func (h *Hub) Broadcast(msgType int, payload []byte) {
	h.broadcast <- Message{msgType, payload}
}

func (h *Hub) BroadcastObject(message interface{}) error {
	bytes, err := json.Marshal(message)
	if err == nil {
		h.Broadcast(websocket.BinaryMessage, bytes)
	}

	return err
}
