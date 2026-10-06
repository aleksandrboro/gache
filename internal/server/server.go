package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/aleksandrboro/gache/internal/aof"
	"github.com/aleksandrboro/gache/internal/command"
	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/storage"
)

type Server struct {
	addr      string
	store     *storage.Store
	router    *command.Router
	aofWriter *aof.AOFWriter
	listener  net.Listener
}

func NewServer(addr string, store *storage.Store, router *command.Router, aofWriter *aof.AOFWriter) *Server {
	return &Server{
		addr:      addr,
		store:     store,
		router:    router,
		aofWriter: aofWriter,
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to create listener. addr: %s\n", s.addr)
	}

	s.listener = listener

	fmt.Printf("server listening on %s\n", s.addr)

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
			writer.Flush()
			continue
		}

		ctx := &command.CommandContext{
			Args:     val.Array,
			Writer:   writer,
			Rewriter: s.aofWriter,
			Store:    s.store,
		}

		if err := s.router.Handle(ctx); err != nil {
			if errors.Is(err, command.ErrQuit) {
				return
			}

			writer.WriteError("ERR failed to handle request")
			writer.Flush()
			continue
		}

		if err := s.aofWriter.WriteCommand(ctx.Args); err != nil {
			if !errors.Is(err, aof.ErrNotWriteCommand) {
				return
			}
		}

		writer.Flush()
	}
}

func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}
