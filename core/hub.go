package core

import (
	log "github.com/sirupsen/logrus"
)

func init() {
	hub = &Hub{
		clients:      make(map[*client]bool),
		addClient:    make(chan *client),
		removeClient: make(chan *client),
	}
}

var hub *Hub

type Hub struct {
	clients      map[*client]bool
	addClient    chan *client
	removeClient chan *client
}

func GetHub() *Hub {
	return hub
}

func (h *Hub) Start() {
	for {
		select {
		case conn := <-h.addClient:
			h.clients[conn] = true
			log.Infof("Added client: %+v", conn)
		case conn := <-h.removeClient:
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				close(conn.send)
				close(conn.monitorNos)
				log.Infof("Removed client: %+v", conn)
			}
		}
	}
}

func (h *Hub) GetClients() []*client {
	keys := make([]*client, len(h.clients))
	i := 0
	for k := range h.clients {
		keys[i] = k
		i++
	}

	return keys
}

func (h *Hub) AddClient(c *client) {
	h.addClient <- c
}
