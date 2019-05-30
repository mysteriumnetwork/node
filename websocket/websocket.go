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
	actions     chan action
}
// send new service instance status to websocket
func (webSocket WebSocket) ServiceUpdateStatusAction(payload interface{}) {
	webSocket.actions <- action{ServiceUpdateStatus, payload}
}

type action struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// add websockets to routes
func AddRoutesForWebSocket(router *httprouter.Router, ws WebSocket) {
	router.GET("/ws/", ws.handler)
}

func (webSocket WebSocket) handler(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	conn, _, _, err := ws.UpgradeHTTP(request, writer)
	if err != nil {
		// handle error
	}
	log.Info("[Websocket] add new connection", conn)
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
				_ = log.Error("[Websocket] read header error", err)
			}

			// Reset writer to write frame with right operation code.
			writer.Reset(conn, state, header.OpCode)
			//
			if _, err = io.Copy(writer, reader); err != nil {
				_ = log.Error("[Websocket] copy error", err)
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

func (webSocket WebSocket) dispatch(action action) {
	bytes, _ := json.Marshal(action)
	for conn := range webSocket.connections {
		log.Info("[Websocket] send to %v data=%v", *conn, string(bytes))
		err := wsutil.WriteServerText(*conn, bytes)

		if err != nil {
			_ = log.Errorf("[Websocket] Error send to %v  data = %v  Error: %v", *conn, string(bytes), err)
		}
	}
}

func (webSocket WebSocket) listenActions() {
	for {
		action := <-webSocket.actions
		if action.Type != "" {
			webSocket.dispatch(action)
		}
	}
}

// create new websocket server
func NewWebSocketServer() WebSocket {
	webSocket := WebSocket{}
	webSocket.connections = make(map[*net.Conn]bool)
	webSocket.actions = make(chan action)
	go webSocket.listenActions()
	Instance = webSocket
	log.Info("[Websocket] Init socket instance")
	return webSocket
}
