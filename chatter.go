package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type connection struct {
	wsc  *websocket.Conn
	cm   *connectionManager
	send chan []byte
}

type connectionManager struct {
	connections      map[*connection]bool
	broadcast        chan []byte
	addConnection    chan *connection
	removeConnection chan *connection
}

func (conn *connection) reader() {
	for {
		_, msg, err := conn.wsc.ReadMessage()
		if err != nil {
			break
		}
		conn.cm.broadcast <- msg
	}
	conn.wsc.Close()
}

func (conn *connection) writer() {
	for msg := range conn.send {
		err := conn.wsc.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			break
		}
	}
	conn.wsc.Close()
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type wsHandler struct {
	cm *connectionManager
}

func (wsh wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// upgrade the connection to a websocket
	wsc, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := &connection{send: make(chan []byte, 256), wsc: wsc, cm: wsh.cm}
	conn.cm.addConnection <- conn
	// ensure connections are closed on program exit
	defer func() { conn.cm.removeConnection <- conn }()
	// start goroutine for writing from this connection
	go conn.writer()
	conn.reader()
}

func newConnectionManager() *connectionManager {
	cm := &connectionManager{
		connections:      make(map[*connection]bool),
		broadcast:        make(chan []byte),
		addConnection:    make(chan *connection),
		removeConnection: make(chan *connection),
	}
	return cm
}

func (cm *connectionManager) run() {
	for {
		select {
		case conn := <-cm.addConnection:
			cm.connections[conn] = true
		case conn := <-cm.removeConnection:
			if _, exists := cm.connections[conn]; exists {
				// remove the connection from the map
				delete(cm.connections, conn)
				// and make sure the channel is closed
				close(conn.send)
			}
		case msg := <-cm.broadcast:
			for conn := range cm.connections {
				select {
				case conn.send <- msg:
				default:
					// remove the connection as something went wrong
					delete(cm.connections, conn)
					close(conn.send)
				}
			}
		}
	}
}

func main() {
	port := flag.Int("port", 3000, "server port")
	dir := flag.String("directory", "app", "client files")
	flag.Parse()

	cm := newConnectionManager()

	go cm.run()

	fs := http.Dir(*dir)
	fileHandler := http.FileServer(fs)
	http.Handle("/", fileHandler)
	http.Handle("/ws", wsHandler{cm: cm})

	log.Printf("Started on port %d\n", *port)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
