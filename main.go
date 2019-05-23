package main

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/googollee/go-socket.io"
	"github.com/julienschmidt/httprouter"
	"log"
	"net"
	"net/http"
	"time"
)

type WebSocket struct {
	Server      *socketio.Server
	connections map[*net.Conn]bool
	//channels    map[string]WebSocketChannel
}

//type WebSocketChannel struct {
//}
//
type WebSocketAction struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func AddRoutesForWebSocket(router *httprouter.Router, ws WebSocket) {
	router.GET("/socket.io/", ws.Handler)
}

func (webSocket WebSocket) Handler(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	conn, _, _, err := ws.UpgradeHTTP(request, writer)
	println("CONNECTED", conn)
	if err != nil {
		// handle error
	}
	webSocket.connections[&conn] = true

	go func() {
		defer func() {
			println("DISCONNECTED", conn)
			delete(webSocket.connections, &conn)
			conn.Close()
		}()

		for {
			msg, op, err := wsutil.ReadClientData(conn)
			if len(msg)>0 {
				println("RECEIVE",string(msg),op)

			}
			if err != nil {
				//println("RECEIVE error",err)
			}
			err = wsutil.WriteServerMessage(conn, op, msg)
			if err != nil {
				// handle error
			}
		}
	}()

}

func (webSocket WebSocket) Send(actionType string, payload interface{}) {
	println("SEND TO ALL",len(webSocket.connections))

	action := WebSocketAction{actionType, payload}
	bytes, _ := json.Marshal(action)
	for conn := range webSocket.connections {
		err := wsutil.WriteServerMessage(*conn, ws.OpText, bytes)
		if err != nil {
			log.Println("WS ERROR SEND MSG ", string(bytes),err)
		}
		println("WS send", actionType, payload, string(bytes))
	}
}

func NewWebSocketServer() WebSocket {
	webSocket := WebSocket{}
	webSocket.connections = make(map[*net.Conn]bool)

	return webSocket
}

//func NewWebSocketServer() WebSocket {
//	ws := WebSocket{}
//	ws.connections = make(map[*socketio.Conn]bool)
//	pt := polling.Default
//
//	wt := websocket.Default
//	wt.CheckOrigin = func(req *http.Request) bool {
//		return true
//	}
//
//	server, err := socketio.NewServer(&engineio.Options{
//		PingInterval: time.Second * 1,
//		PingTimeout:  time.Second * 3,
//		//PingTimeout: 15000,
//		Transports: []transport.Transport{
//			pt,
//			wt,
//		},
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	ws.Server = server
//	ws.Server.OnConnect("/", func(s socketio.Conn) error {
//		s.SetContext("")
//		ws.connections[&s] = true
//		fmt.Println("WS connected:", s.ID())
//		return nil
//	})
//	ws.Server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
//		fmt.Println("WS notice:", msg)
//		s.Emit("reply", "have "+msg)
//	})
//	ws.Server.OnError("/", func(e error) {
//		fmt.Println("WS meet error:", e)
//	})
//	ws.Server.OnDisconnect("/", func(s socketio.Conn, msg string) {
//		fmt.Println("WS closed", msg)
//		delete(ws.connections, &s)
//	})
//	server.OnEvent("/", "bye", func(s socketio.Conn) string {
//		last := s.Context().(string)
//		s.Emit("bye", last)
//		delete(ws.connections, &s)
//		s.Close()
//		return last
//	})
//	go ws.Server.Serve()
//	//http.Handle("/socket.io/", ws.Server)
//	//log.Println(fmt.Sprintf("Websockets starting at"+
//	//	""+
//	//	""+
//	//	""+
//	//	" localhost: :%v...", "40501"))
//	//go http.ListenAndServe(":40501", nil)
//	i := 0
//	setInterval(func() {
//		ws.Send("socket/on/TEST", fmt.Sprintf("aaa %v", i))
//		i++
//	}, 5000, true)
//
//	return ws
//
//}

//
//
//import (
//	"fmt"
//	"github.com/mysteriumnetwork/node/core/node"
//	"log"
//	"net/http"
//
//	"github.com/gorilla/websocket"
//)
//
//type Message struct {
//	Action  string `json:"action"`
//	Payload string `json:"payload"`
//}
//
//type WebSocket struct {
//	//Broadcast chan Message
//	clients  map[*websocket.Conn]bool
//	upgrader websocket.Upgrader
//}
//
//func (webSocket WebSocket) handleConnections(w http.ResponseWriter, r *http.Request) {
//	// Upgrade initial GET request to a websocket
//	ws, err := webSocket.upgrader.Upgrade(w, r, nil)
//	println("create client", ws)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Make sure we close the connection when the function returns
//	defer func() {
//		delete(webSocket.clients, ws)
//		println("delete client", ws)
//		ws.Close()
//	}()
//	// Register our new client
//	webSocket.clients[ws] = true
//	for {
//		var msg Message
//		// Read in a new message as JSON and map it to a Message object
//		err := ws.ReadJSON(&msg)
//		if err != nil {
//			log.Printf("error: %v", err)
//			break
//		}
//		// Send the newly received message to the broadcast channel
//		//webSocket.Broadcast <- msg
//	}
//}
//func (webSocket WebSocket) SendMessage(msg Message) {
//	for client := range webSocket.clients {
//		err := client.WriteJSON(msg)
//		if err != nil {
//			log.Printf("error: %v", err)
//			client.Close()
//			delete(webSocket.clients, client)
//		}
//	}
//}
//
//func NewWebSocket(nodeOptions node.Options) WebSocket {
//	nodeOptions.WebSocketPort = 4060
//	println("!!!!!!")
//	ws := WebSocket{}
//	ws.clients = make(map[*websocket.Conn]bool)
//	//ws.Broadcast = make(chan Message)
//	ws.upgrader = websocket.Upgrader{}
//
//	http.HandleFunc("/", ws.handleConnections)
//	log.Println(fmt.Sprintf("WebSocket started on :%v", nodeOptions.WebSocketPort))
//	err := http.ListenAndServe(":4060", nil)
//	if err != nil {
//		log.Fatal("WebSocket start error: ", err)
//	}
//	return ws
//}
func setInterval(someFunc func(), milliseconds int, async bool) chan bool {

	// How often to fire the passed in function
	// in milliseconds
	interval := time.Duration(milliseconds) * time.Millisecond

	// Setup the ticket and the channel to signal
	// the ending of the interval
	ticker := time.NewTicker(interval)
	clear := make(chan bool)

	// Put the selection in a go routine
	// so that the for loop is none blocking
	go func() {
		for {

			select {
			case <-ticker.C:
				if async {
					// This won't block
					go someFunc()
				} else {
					// This will block
					someFunc()
				}
			case <-clear:
				ticker.Stop()
				return
			}

		}
	}()

	// We return the channel so we can pass in
	// a value to it to clear the interval
	return clear

}

func main() {
	wss := NewWebSocketServer()
	router := httprouter.New()
	router.HandleMethodNotAllowed = true
	AddRoutesForWebSocket(router, wss)
	i := 0
	setInterval(func() {

		wss.Send("socket/on/TEST", i)
		i++
	}, 5000, true)
	http.ListenAndServe(":8080", router)


}
