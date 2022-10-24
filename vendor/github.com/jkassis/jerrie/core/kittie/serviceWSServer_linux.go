package kittie

import (
	context "context"
	"net"
	"net/http"
	"reflect"
	"sync"
	"syscall"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/jkassis/jerrie/core"
	"golang.org/x/sys/unix"
)

// WSServer polls for events on sockets and forwards to registered handlers
type WSServer struct {
	fd          int
	connections map[int]connHandler
	lock        *sync.RWMutex
	playing     bool
}

type connHandler struct {
	conn    net.Conn
	handler Handler
}

// MakeWSServer : A FS Event Poller
// http://man7.org/linux/man-pages/man7/epoll.7.html
func MakeWSServer() (*WSServer, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &WSServer{
		fd:          fd,
		lock:        &sync.RWMutex{},
		connections: make(map[int]connHandler),
	}, nil
}

// UpgradeToWS establishes a websocket with the caller and forwards data to the handler
func (s *WSServer) UpgradeToWS(req *http.Request, res http.ResponseWriter, handler Handler) error {
	// Upgrade connection
	conn, _, _, err := ws.UpgradeHTTP(req, res)
	if err != nil {
		return err
	}
	if err := s.add(connHandler{conn: conn, handler: handler}); err != nil {
		core.Log.Info("Failed to add connection %v", err)
		conn.Close()
	}

	return nil
}

func (s *WSServer) add(connHandler connHandler) error {
	// Extract file descriptor associated with the connection
	fd := websocketFD(connHandler.conn)
	err := unix.EpollCtl(s.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.connections[fd] = connHandler
	if len(s.connections)%100 == 0 {
		core.Log.Info("Total number of connections: %v", len(s.connections))
	}
	return nil
}

func (s *WSServer) remove(connHandler connHandler) error {
	fd := websocketFD(connHandler.conn)
	err := unix.EpollCtl(s.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.connections, fd)
	if len(s.connections)%100 == 0 {
		core.Log.Info("Total number of connections: %v", len(s.connections))
	}
	return nil
}

// Play starts the WSServer
func (s *WSServer) Play() error {
	if s.playing {
		return nil
	}
	s.playing = true

	for {
		if !s.playing {
			return nil
		}

		connHandlers, err := s.wait()
		if err != nil {
			core.Log.Info("Failed to epoll wait %v", err)
			continue
		}
		for _, connHandler := range connHandlers {
			if connHandler.conn == nil {
				break
			}
			if msg, _, err := wsutil.ReadClientData(connHandler.conn); err != nil {
				if err := s.remove(connHandler); err != nil {
					core.Log.Error("Failed to remove %v", err)
				}
				connHandler.conn.Close()
			} else {
				if out, err := connHandler.handler(context.Background(), msg); err != nil {
					core.Log.Error(err)
				} else {
					err = wsutil.WriteServerMessage(connHandler.conn, ws.OpBinary, out)
					if err != nil {
						// handle error
					}
				}

				// This is commented out since in demo usage, stdout is showing messages sent from > 1M connections at very high rate
				//core.Log.Info("msg: %s", string(msg))
			}
		}
	}
}

// Stop stops the service
func (s *WSServer) Stop() error {
	if !s.playing {
		return nil
	}
	s.playing = false

	return nil
}

func (s *WSServer) wait() ([]connHandler, error) {
	events := make([]unix.EpollEvent, 100)
	n, err := unix.EpollWait(s.fd, events, 100)
	if err != nil {
		return nil, err
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	var connHandlers []connHandler
	for i := 0; i < n; i++ {
		conn := s.connections[int(events[i].Fd)]
		connHandlers = append(connHandlers, conn)
	}
	return connHandlers, nil
}

// returns a file descriptor representing the websocket
func websocketFD(conn net.Conn) int {
	//tls := reflect.TypeOf(conn.UnderlyingConn()) == reflect.TypeOf(&tls.Conn{})
	// Extract the file descriptor associated with the connection
	//connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	//if tls {
	//	tcpConn = reflect.Indirect(tcpConn.Elem())
	//}
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")

	return int(pfdVal.FieldByName("Sysfd").Int())
}
