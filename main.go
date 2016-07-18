package main

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/oliverbestmann/go-websocket-publish/ws"
	"io/ioutil"
	"net/http"
)

type J map[string]interface{}

func main() {
	clientSide := mux.NewRouter()

	tokenManager := ws.NewTokenManager()
	registrar := ws.NewRegistrar()

	clientSide.Path("/streams/{stream}/tokens/{token}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		withStream(w, req, registrar, func(streamId ws.StreamId, hub *ws.Hub) {
			token := ws.TokenFromString(vars["token"])
			if ! tokenManager.Allowed(streamId, token) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			handleClientWebSocket(hub, token, w, req)
		})
	})

	serverSide := mux.NewRouter()

	serverSide.Path("/streams").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		streams := registrar.GetStreams()
		WriteJSON(w, streams)
	})

	serverSide.Path("/streams/{stream}").Methods("DELETE").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		streamId := vars["stream"]

		// TODO check for access rights and stuff

		if !registrar.Close(streamId) {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	serverSide.Path("/streams/{stream}/tokens").Methods("POST").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		withStream(w, req, registrar, func(streamId ws.StreamId, hub *ws.Hub) {
			token := ws.RandomToken()
			tokenManager.Allow(streamId, token)

			WriteJSON(w, J{"stream": streamId, "token": token.String()})
		})
	})

	serverSide.Path("/streams/{stream}/tokens/{token}").Methods("DELETE").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		withStream(w, req, registrar, func(streamId ws.StreamId, hub *ws.Hub) {
			token := ws.TokenFromString(vars["token"])

			tokenManager.Revoke(streamId, token)
			hub.RevokeToken(token)
		})
	})

	serverSide.Path("/streams/{stream}/publish").Methods("GET").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		streamId := vars["stream"]

		hub := registrar.GetOrCreateHub(streamId)
		if hub == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		handleSenderWebSocket(hub, w, req)
	})

	go func() {
		logger := logrus.StandardLogger()
		logrus.Fatal(http.ListenAndServe(":8080",
			handlers.LoggingHandler(logger.Writer(),
				handlers.RecoveryHandler()(serverSide))))
	}()

	logger := logrus.StandardLogger()
	logrus.Fatal(http.ListenAndServe(":8081",
		handlers.LoggingHandler(logger.Writer(),
			handlers.RecoveryHandler()(clientSide))))
}

func WriteJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}

type WithStreamHandler func(ws.StreamId, *ws.Hub)

func withStream(w http.ResponseWriter, req *http.Request, registrar *ws.Registrar, handler WithStreamHandler) {
	vars := mux.Vars(req)
	streamId := ws.StreamId(vars["stream"])
	hub := registrar.GetExistingHub(string(streamId))
	if hub != nil {
		handler(ws.StreamId(streamId), hub)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func handleClientWebSocket(hub *ws.Hub, token ws.Token, writer http.ResponseWriter, request *http.Request) {
	socket, err := websocket.Upgrade(writer, request, nil, 0, 0)
	if err != nil {
		logrus.WithError(err).Warn("Could not upgrade websocket")
		return
	}

	hub.HandleConnection(token, socket)
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
