package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

type hubConnection struct {
	socket *websocket.Conn
	send   chan Message
}

func (conn *hubConnection) writeLoop() {
	defer conn.socket.Close()

	for message := range conn.send {
		// write the message to the socket
		if err := conn.socket.WriteMessage(message.Type, message.Payload); err != nil {
			logrus.WithError(err).Warn("Could not write message to websocket")
			break
		}
	}
}

/**
 * Reads all the messages of a websocket until it is closed.
 */
func (conn *hubConnection) readLoop() {
	for {
		if _, _, err := conn.socket.NextReader(); err != nil {
			logrus.WithError(err).Warn("Error reading websocket, closing now.")
			conn.socket.Close()
			break
		}
	}
}
