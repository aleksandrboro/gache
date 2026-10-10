package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/aleksandrboro/gache/internal/aof"
	"github.com/aleksandrboro/gache/internal/command"
	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/pubsub"
	"github.com/aleksandrboro/gache/internal/storage"
)

type Server struct {
	addr      string
	store     *storage.Store
	router    *command.Router
	aofWriter *aof.AOFWriter
	hub       *pubsub.Hub
	listener  net.Listener
}

func NewServer(addr string, store *storage.Store, router *command.Router, aofWriter *aof.AOFWriter, hub *pubsub.Hub) *Server {
	return &Server{
		addr:      addr,
		store:     store,
		router:    router,
		aofWriter: aofWriter,
		hub:       hub,
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

	sub := pubsub.NewSubscriber()
	var writeMu sync.Mutex

	defer func() {
		close(sub.Done)
		if s.hub != nil {
			s.hub.UnsubscribeAll(sub)
		}
	}()

	// Forwarding goroutine: reads from MsgChan and writes to TCP
	go func() {
		for {
			select {
			case msg := <-sub.MsgChan:
				writeMu.Lock()
				writer.WriteArray([]protocol.RESPValue{
					{Type: protocol.BulkString, Str: msg.Type},
					{Type: protocol.BulkString, Str: msg.Channel},
					{Type: protocol.BulkString, Str: msg.Data},
				})
				writer.Flush()
				writeMu.Unlock()
			case <-sub.Done:
				return
			}
		}
	}()

	for {
		val, err := parser.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			writeMu.Lock()
			writer.WriteError(err.Error())
			writer.Flush()
			writeMu.Unlock()
			continue
		}

		if val.Type != protocol.Array || len(val.Array) == 0 {
			writeMu.Lock()
			writer.WriteError("ERR invalid command format")
			writer.Flush()
			writeMu.Unlock()
			continue
		}

		ctx := &command.CommandContext{
			Args:     val.Array,
			Writer:   writer,
			Rewriter: s.aofWriter,
			Store:    s.store,
			Hub:      s.hub,
			Sub:      sub,
			WriteMu:  &writeMu,
		}

		if err := s.router.Handle(ctx); err != nil {
			if errors.Is(err, command.ErrQuit) {
				return
			}

			writeMu.Lock()
			writer.WriteError("ERR failed to handle request")
			writer.Flush()
			writeMu.Unlock()
			continue
		}

		if err := s.aofWriter.WriteCommand(ctx.Args); err != nil {
			if !errors.Is(err, aof.ErrNotWriteCommand) {
				return
			}
		}

		writeMu.Lock()
		writer.Flush()
		writeMu.Unlock()
	}
}

func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}
