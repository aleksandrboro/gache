package command

import (
	"fmt"
	"strings"
)

type Handler func(ctx *CommandContext) error

type Router struct {
	handlers map[string]Handler
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]Handler),
	}
}

func (r *Router) RegisterCommands() {
	r.Register("PING", cmdPing)
	r.Register("ECHO", cmdEcho)
	r.Register("SET", cmdSet)
	r.Register("GET", cmdGet)
	r.Register("DEL", cmdDel)
	r.Register("EXISTS", cmdExists)
	r.Register("INCR", cmdIncr)
	r.Register("INCRBY", cmdIncrBy)
	r.Register("DECR", cmdDecr)
	r.Register("DECRBY", cmdDecrBy)
	r.Register("QUIT", cmdQuit)
	r.Register("MSET", cmdMSet)
	r.Register("MGET", cmdMGet)
	r.Register("EXPIRE", cmdExpire)
	r.Register("PEXPIRE", cmdPExpire)
	r.Register("TTL", cmdTTL)
	r.Register("PTTL", cmdPTTL)
	r.Register("PERSIST", cmdPersist)
}

func (r *Router) Register(name string, handler Handler) {
	r.handlers[strings.ToUpper(name)] = handler
}

func (r *Router) Handle(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return ctx.Writer.WriteError("ERR empty command")
	}

	cmdName := strings.ToUpper(ctx.Args[0].Str)

	handler, ok := r.handlers[cmdName]
	if !ok {
		return ctx.Writer.WriteError(fmt.Sprintf("ERR unknown command '%s'", ctx.Args[0].Str))
	}

	return handler(ctx)
}
