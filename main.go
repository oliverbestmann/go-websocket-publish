package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"sync"
)

type Registrar struct {
	lock    sync.RWMutex
	streams map[string]*Hub
}

func NewRegistrar() *Registrar {
	return &Registrar{
		lock:    sync.RWMutex{},
		streams: make(map[string]*Hub),
	}
}

func (r *Registrar) GetExistingHub(streamId string) *Hub {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.streams[streamId]
}

func (r *Registrar) GetOrCreateHub(streamId string) *Hub {
	r.lock.Lock()
	defer r.lock.Unlock()

	hub, _ := r.streams[streamId]
	if hub == nil {
		hub = NewHub()
		r.streams[streamId] = hub

		go hub.MainLoop()
	}

	return hub
}

func (r *Registrar) Close(streamId string) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	if hub := r.streams[streamId]; hub != nil {
		delete(r.streams, streamId)
		hub.RequestShutdown()
		return true

	} else {
		return false
	}
}

func main() {
	router := mux.NewRouter()

	registrar := NewRegistrar()

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

		if !registrar.Close(streamId) {
			w.WriteHeader(http.StatusNotFound)
		}
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
