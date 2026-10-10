package command

import (
	"sync"

	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/pubsub"
	"github.com/aleksandrboro/gache/internal/storage"
)

type CommandContext struct {
	Args     []protocol.RESPValue
	Writer   *protocol.Writer
	Store    *storage.Store
	Rewriter Rewriter
	Hub      *pubsub.Hub
	Sub      *pubsub.Subscriber
	WriteMu  *sync.Mutex
}

type Rewriter interface {
	Rewrite(store *storage.Store) error
}
