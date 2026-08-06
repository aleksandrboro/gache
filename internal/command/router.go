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
