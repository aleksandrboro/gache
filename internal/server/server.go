package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/aleksandrboro/gache/internal/command"
	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/storage"
)

type Server struct {
	addr     string
	store    *storage.Store
	router   *command.Router
	listener net.Listener
}

func NewServer(addr string, store *storage.Store, router *command.Router) *Server {
	return &Server{
		addr:   addr,
		store:  store,
		router: router,
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to create listener. addr: %s", s.addr)
	}

	s.listener = listener

	fmt.Printf("server listening on %s", s.addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			listener.Close()
			return err
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	parser := protocol.NewParser(r)
	writer := protocol.NewWriter(w)

	for {
		val, err := parser.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			writer.WriteError(err.Error())
			writer.Flush()
			continue
		}

		if val.Type != protocol.Array || len(val.Array) == 0 {
			writer.WriteError("ERR invalid command format")
			continue
		}

		ctx := 
	}
}
