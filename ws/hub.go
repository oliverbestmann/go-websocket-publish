package ws

import (
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

// number of packages to buffer for each client.
const SendChannelSize = 8

type Token uuid.UUID

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
	broadcast       chan Message

	// all the currently registered connections
	connections     map[*hubConnection]none

	// register requests
	register        chan *hubConnection

	// unregister requests
	unregister      chan *hubConnection

	// unregister by token requests
	unregisterToken chan Token
}

type none struct{}

func NewHub() *Hub {
	return &Hub{
		broadcast:       make(chan Message, 16),
		register:        make(chan *hubConnection),
		unregister:      make(chan *hubConnection),
		unregisterToken: make(chan Token),
		connections:     make(map[*hubConnection]none),
	}
}

/**
 * Main loop for this hub. This method blocks forever, so it is best to
 * call it in a go-routine.
 */
func (h *Hub) MainLoop() {
	loop:
	for {
		select {
		case conn := <-h.register:
			h.connections[conn] = none{}

		case token := <-h.unregisterToken:
			for conn := range h.connections {
				if conn.token == token {
					delete(h.connections, conn)
					close(conn.send)
				}
			}

		case conn := <-h.unregister:
			if _, ok := h.connections[conn]; ok {
				delete(h.connections, conn)
				close(conn.send)
			}

		case message := <-h.broadcast:
			if message.Type == websocket.CloseMessage {
				break loop
			}

			for conn := range h.connections {
				// put the message on the send-queue for this connection
				select {
				case conn.send <- message:
				default:
					// couldn't write to the queue. looks like the client has
					// problems receiving the messages. we'll just empty the queue.
				  // and add the message back again
					conn.drainQueue()
					conn.send <- message
				}
			}
		}
	}

	// close all sockets
	for conn := range h.connections {
		delete(h.connections, conn)
		close(conn.send)
	}
}

func (h *Hub) unregisterConnection(conn *hubConnection) {
	h.unregister <- conn
}

/**
 * Handles the given websocket connection. This call blocks
 * until the connection was closed.
 */
func (h *Hub) HandleConnection(token Token, socket *websocket.Conn) {
	// make a new connection object
	conn := &hubConnection{
		socket: socket,
		send:   make(chan Message, SendChannelSize),
		token: token,
	}

	// register this connection with the hub
	h.register <- conn
	defer h.unregisterConnection(conn)

	// forward writes and consume the read end too.
	go conn.writeLoop()
	conn.readLoop()
}

func (h *Hub) RevokeToken(token Token) {
	h.unregisterToken <- token
}

func (h *Hub) RequestShutdown() {
	h.Broadcast(websocket.CloseMessage, []byte{})
}

func (h *Hub) Broadcast(msgType int, payload []byte) {
	h.broadcast <- Message{msgType, payload}
}

func RandomToken() Token {
	return Token(uuid.NewV4())
}

func TokenFromString(str string) Token {
	return Token(uuid.FromStringOrNil(str))
}

func (t Token) String() string {
	return uuid.UUID(t).String()
}
