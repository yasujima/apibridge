package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type group struct {
	bridge  chan []byte
	join    chan *client
	leave   chan *client
	clients map[*client]bool
}

func (r *group) run() {
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
		case client := <-r.leave:
			delete(r.clients, client)
			close(client.send)
		case msg := <-r.bridge:
			for client := range r.clients {
				select {
				case client.send <- msg:
				// send message ##
				default:
					delete(r.clients, client)
					close(client.send)
				}
			}
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  socketBufferSize,
	WriteBufferSize: socketBufferSize}

func (r *group) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}
