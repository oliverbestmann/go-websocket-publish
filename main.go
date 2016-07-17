package main

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"github.com/oliverbestmann/go-websocket-publish/ws"
)

func main() {
	router := mux.NewRouter()

	registrar := ws.NewRegistrar()

	router.Path("/streams").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		streams := registrar.GetStreams()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(streams)
	})

	router.Path("/streams/{stream}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		streamId := vars["stream"]

		hub := registrar.GetExistingHub(streamId)
		if hub == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		handleClientWebSocket(hub, w, req)
	})

	router.Path("/streams/{stream}/publish").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		streamId := vars["stream"]

		// TODO check for access rights and stuff

		hub := registrar.GetOrCreateHub(streamId)
		if hub == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		handleSenderWebSocket(hub, w, req)
	})

	router.Path("/streams/{stream}").Methods("DELETE").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		streamId := vars["stream"]

		// TODO check for access rights and stuff

		if !registrar.Close(streamId) {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	panic(http.ListenAndServe(":8081",
		handlers.LoggingHandler(logrus.StandardLogger().Writer(),
			handlers.RecoveryHandler()(router))))
}

func handleClientWebSocket(hub *ws.Hub, writer http.ResponseWriter, request *http.Request) {
	socket, err := websocket.Upgrade(writer, request, nil, 0, 0)
	if err != nil {
		logrus.WithError(err).Warn("Could not upgrade websocket")
		return
	}

	hub.HandleConnection(socket)
}

func handleSenderWebSocket(hub *ws.Hub, writer http.ResponseWriter, request *http.Request) {
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

		} else if msgType != websocket.CloseMessage {
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
