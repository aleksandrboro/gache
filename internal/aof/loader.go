package aof

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/aleksandrboro/gache/internal/command"
	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/storage"
)

func LoadAOF(filename string, store *storage.Store, router *command.Router) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}
	defer file.Close()

	parser := protocol.NewParser(bufio.NewReader(file))

	dummyBuf := bytes.Buffer{}
	dummyWriter := protocol.NewWriter(bufio.NewWriter(&dummyBuf))

	for {
		value, err := parser.Parse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		ctx := &command.CommandContext{
			Args:   value.Array,
			Writer: dummyWriter,
			Store:  store,
		}

		if err := router.Handle(ctx); err != nil {
			return err
		}
	}
}
