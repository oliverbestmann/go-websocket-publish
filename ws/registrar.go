package ws

import (
	"sync"
	"time"
)

type Stream struct {
	Id      string    `json:"id"`
	Created time.Time `json:"created"`

	hub *Hub
}

type Registrar struct {
	lock    sync.RWMutex
	streams map[string]*Stream
}

func NewRegistrar() *Registrar {
	return &Registrar{
		lock:    sync.RWMutex{},
		streams: make(map[string]*Stream),
	}
}

func (r *Registrar) GetExistingHub(streamId string) *Hub {
	r.lock.RLock()
	defer r.lock.RUnlock()

	stream := r.streams[streamId]
	if stream == nil {
		return nil
	}

	return stream.hub
}

func (r *Registrar) GetOrCreateHub(streamId string) *Hub {
	r.lock.Lock()
	defer r.lock.Unlock()

	stream, _ := r.streams[streamId]
	if stream == nil {
		stream = &Stream{
			Id:      streamId,
			Created: time.Now(),
			hub:     NewHub(),
		}

		r.streams[streamId] = stream

		go stream.hub.MainLoop()
	}

	return stream.hub
}

func (r *Registrar) GetStreams() []Stream {
	r.lock.Lock()
	defer r.lock.Unlock()

	// get a copy of the streams id.
	streams := make([]Stream, 0, len(r.streams))
	for _, stream := range r.streams {
		streams = append(streams, *stream)
	}

	return streams
}

func (r *Registrar) Close(streamId string) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	if stream := r.streams[streamId]; stream != nil {
		delete(r.streams, streamId)
		stream.hub.RequestShutdown()
		return true

	} else {
		return false
	}
}
