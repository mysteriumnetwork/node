package websocket

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/googollee/go-socket.io"
	"github.com/julienschmidt/httprouter"
	"io"
	"net"
	"net/http"
)

const ServiceUpdateStatus = "server/SERVICE_UPDATE_STATUS"

var Instance WebSocket

type WebSocket struct {
	Server      *socketio.Server
	connections map[*net.Conn]bool
	Actions     chan Action
}

func (webSocket WebSocket) ServiceUpdateStatusAction(payload interface{}) {
	webSocket.Actions <- Action{ServiceUpdateStatus, payload}
}

type Action struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func AddRoutesForWebSocket(router *httprouter.Router, ws WebSocket) {
	router.GET("/ws/", ws.Handler)
}

func (webSocket WebSocket) Handler(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	conn, _, _, err := ws.UpgradeHTTP(request, writer)
	if err != nil {
		// handle error
	}
	log.Info("[Websocket] ADD CONN", conn)
	webSocket.connections[&conn] = true
	go func() {
		defer func() {
			delete(webSocket.connections, &conn)
			_ = log.Error("[Websocket] delete connection. reason: DEFER CLOSE")
			conn.Close()
		}()

		var (
			state  = ws.StateServerSide
			reader = wsutil.NewReader(conn, state)
			writer = wsutil.NewWriter(conn, state, ws.OpText)
		)
		for {
			header, err := reader.NextFrame()
			if err != nil {
				// handle error
			}

			// Reset writer to write frame with right operation code.
			writer.Reset(conn, state, header.OpCode)
			//
			if _, err = io.Copy(writer, reader); err != nil {
				// handle error
			}
			if err = writer.Flush(); err != nil {
				// handle error
			}
			if header.OpCode == ws.OpClose {
				_ = log.Error("[Websocket] delete connection. reason: CLOSE")
				delete(webSocket.connections, &conn)
			}
		}
	}()

}

func (webSocket WebSocket) Dispatch(action Action) {
	bytes, _ := json.Marshal(action)
	for conn := range webSocket.connections {
		log.Info("[Websocket] send to %v data=%v", *conn,string(bytes))
		err := wsutil.WriteServerText(*conn, bytes)

		if err != nil {
			_ = log.Errorf("[Websocket] Error send to %v  data = %v  Error: %v", *conn, string(bytes), err)
		}
	}
}

func (webSocket WebSocket) ListenActions() {
	for {
		action := <-webSocket.Actions
		if action.Type != "" {
			webSocket.Dispatch(action)
		}
	}
}

type Test struct {
	Id    int    `json:"id"`
	State string `json:"state"`
	Some  string
}

func NewWebSocketServer() WebSocket {
	webSocket := WebSocket{}
	webSocket.connections = make(map[*net.Conn]bool)
	webSocket.Actions = make(chan Action)
	go webSocket.ListenActions()
	Instance = webSocket
	log.Info("[Websocket] Init socket instance")
	return webSocket
}