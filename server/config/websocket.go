package config

import (
	"net/http"

	"github.com/gorilla/websocket"
)

func GetWebsocketUpgrader() websocket.Upgrader {
	instance := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	return instance
}
