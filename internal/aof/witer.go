package aof

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aleksandrboro/gache/internal/protocol"
)

const (
	alwaysPolicy   = "always"
	everySecPolicy = "everysec"
	noPolicy       = "no"
)

var (
	writeCommands = map[string]struct{}{
		"SET":     struct{}{},
		"DEL":     struct{}{},
		"MSET":    struct{}{},
		"INCR":    struct{}{},
		"INCRBY":  struct{}{},
		"DECR":    struct{}{},
		"DECRBY":  struct{}{},
		"LPUSH":   struct{}{},
		"RPUSH":   struct{}{},
		"LPOP":    struct{}{},
		"RPOP":    struct{}{},
		"HSET":    struct{}{},
		"HDEL":    struct{}{},
		"SADD":    struct{}{},
		"SREM":    struct{}{},
		"ZADD":    struct{}{},
		"ZREM":    struct{}{},
		"EXPIRE":  struct{}{},
		"PEXPIRE": struct{}{},
		"PERSIST": struct{}{},
	}

	ErrNotWriteCommand = errors.New("Command isn't for writing")
)

type AOFWriter struct {
	file        *os.File
	writer      *bufio.Writer
	mu          sync.Mutex
	fsyncPolicy string
}

func NewAOFWriter(ctx context.Context, filename, fsyncPolicy string) (*AOFWriter, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriter(file)

	if fsyncPolicy == everySecPolicy {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					file.Sync()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return &AOFWriter{
		file:        file,
		writer:      writer,
		fsyncPolicy: fsyncPolicy,
	}, nil
}

func (w *AOFWriter) WriteCommand(args []protocol.RESPValue) error {
	if _, ok := writeCommands[strings.ToUpper(args[0].Str)]; !ok {
		return ErrNotWriteCommand
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	buf := bytes.Buffer{}
	writer := protocol.NewWriter(bufio.NewWriter(&buf))

	if err := writer.WriteArray(args); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if _, err := w.file.WriteString(buf.String()); err != nil {
		return err
	}

	if w.fsyncPolicy == alwaysPolicy {
		if err := w.file.Sync(); err != nil {
			return err
		}
	}

	return nil
}

func (w *AOFWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Sync(); err != nil {
		return err
	}

	if err := w.file.Close(); err != nil {
		return err
	}

	return nil
}
