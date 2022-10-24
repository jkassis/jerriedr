package kittie

import (
	context "context"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jkassis/jerrie/core"
)

// UpgradeToWS upgrades the HTTP request to a websocket
func UpgradeToWS(ctx context.Context, req *http.Request, res http.ResponseWriter, handler Handler) (*websocket.Conn, error) {
	// Upgrade connection
	upgrader := websocket.Upgrader{
		CheckOrigin: func(req *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		return nil, err
	}

	go func() {
		defer core.SentryRecover("WSServer.UpgradeToWS")
		for {
			_, reader, err := conn.NextReader()
			if err != nil {
				conn.Close()
				return
			}
			writer, err := conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				conn.Close()
				return
			}

			var handlerRes []byte
			req, err := ioutil.ReadAll(reader)
			if err == nil {
				handlerRes, err = handler(ctx, req)
			}
			if err != nil {
				writer.Write([]byte(err.Error()))
				core.Log.Error(err)
				return
			}
			writer.Write(handlerRes)
			if err = writer.Close(); err != nil {
				core.Log.Error(err)
				return
			}
		}
	}()

	return conn, nil
}
