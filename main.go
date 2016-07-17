package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
)

func main() {
	router := mux.NewRouter()

	hub := NewHub()
	go hub.MainLoop()

	router.Path("/subscribe").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleClientWebSocket(hub, w, req)
	})

	router.Path("/publish").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handleSenderWebSocket(hub, w, req)
	})

	panic(http.ListenAndServe(":8081",
		handlers.LoggingHandler(logrus.StandardLogger().Writer(),
			handlers.RecoveryHandler()(router))))
}

func handleClientWebSocket(hub *Hub, writer http.ResponseWriter, request *http.Request) {
	socket, err := websocket.Upgrade(writer, request, nil, 0, 0)
	if err != nil {
		logrus.WithError(err).Warn("Could not upgrade websocket")
		return
	}

	hub.HandleConnection(socket)
}

func handleSenderWebSocket(hub *Hub, writer http.ResponseWriter, request *http.Request) {
	socket, err := websocket.Upgrade(writer, request, nil, 0, 0)
	if err != nil {
		logrus.WithError(err).Warn("Could not upgrade websocket")
		return
	}

	for {
		if msgType, msgContent, err := socket.NextReader(); err != nil {
			logrus.WithError(err).Warn("Error reading sender websocket, closing now.")
			socket.Close()
			break

		} else {
			payload, err := ioutil.ReadAll(msgContent)
			if err != nil {
				logrus.WithError(err).Warn("Could not read a complete data frame from sender.")
				continue
			}

			logrus.WithField("type", msgType).
				WithField("bytes", len(payload)).
				Info("Received a dataframe from the server")

			// forward message to all registered clients!
			hub.Broadcast(msgType, payload)
		}
	}
}
