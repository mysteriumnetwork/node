/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package websocket

import (
	"encoding/json"
	"io"
	"net"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	socketio "github.com/googollee/go-socket.io"
	"github.com/julienschmidt/httprouter"
)

const serviceUpdateStatus = "server/SERVICE_UPDATE_STATUS"

// WebSocket struct
type WebSocket struct {
	Server      *socketio.Server
	connections map[*net.Conn]bool
	actions     chan action
	isListen    bool
}

// ServiceUpdateStatusAction - send new service instance status to websocket
func (webSocket *WebSocket) ServiceUpdateStatusAction(payload interface{}) {
	webSocket.actions <- action{serviceUpdateStatus, payload}
}

type action struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// AddRoutesForWebSocket - add websockets to routes
func AddRoutesForWebSocket(router *httprouter.Router, ws WebSocket) {
	router.GET("/ws/", ws.handler)
}

func (webSocket *WebSocket) handler(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	conn, _, _, err := ws.UpgradeHTTP(request, writer)
	if err != nil {
		_ = log.Error("[Websocket] UpgradeHTTP error", err)
	}
	log.Info("[Websocket] add new connection", conn)

	webSocket.connections[&conn] = true
	if !webSocket.isListen {
		go webSocket.listenActions()
	}
	go func() {
		defer func() {
			delete(webSocket.connections, &conn)
			_ = log.Error("[Websocket] delete connection. reason: DEFER CLOSE")
			_ = conn.Close()
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

func (webSocket *WebSocket) dispatch(action action) {
	bytes, _ := json.Marshal(action)
	for conn := range webSocket.connections {
		log.Info("[Websocket] send to %v data=%v", *conn, string(bytes))
		err := wsutil.WriteServerText(*conn, bytes)

		if err != nil {
			_ = log.Errorf("[Websocket] Error send to %v  data = %v  Error: %v", *conn, string(bytes), err)
		}
	}
}

func (webSocket *WebSocket) listenActions() {
	if webSocket.isListen {
		return
	}
	webSocket.isListen = true
	for {
		println("infinity -", len(webSocket.connections))
		if len(webSocket.connections) == 0 {
			break
		}
		action := <-webSocket.actions

		if action.Type != "" {
			webSocket.dispatch(action)
		}
	}
	webSocket.isListen = false
	return
}

// NewWebSocketServer - create new websocket server
func NewWebSocketServer() WebSocket {
	webSocket := WebSocket{}
	webSocket.connections = make(map[*net.Conn]bool)
	webSocket.actions = make(chan action)
	log.Info("[Websocket] Init socket instance")
	return webSocket
}
