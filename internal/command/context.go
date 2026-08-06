package command

import (
	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/storage"
)

type CommandContext struct {
	Args   []protocol.RESPValue
	Writer *protocol.Writer
	Store  *storage.Store
}
