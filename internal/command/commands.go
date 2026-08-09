package command

import (
	"errors"
	"strconv"

	"github.com/aleksandrboro/gache/internal/protocol"
)

var ErrQuit = errors.New("quit")

func cmdPing(ctx *CommandContext) error {
	if len(ctx.Args) == 1 {
		return ctx.Writer.WriteSimpleString("PONG")
	}

	if len(ctx.Args) == 2 {
		return ctx.Writer.WriteBulkString(ctx.Args[1].Str)
	}

	return ctx.Writer.WriteError("ERR wrong number of arguments for 'ping' command")
}

func cmdEcho(ctx *CommandContext) error {
	if len(ctx.Args) == 2 {
		return ctx.Writer.WriteBulkString(ctx.Args[1].Str)
	}

	return ctx.Writer.WriteError("ERR wrong number of arguments for 'echo' command")
}

func cmdSet(ctx *CommandContext) error {
	if len(ctx.Args) == 3 {
		key := ctx.Args[1].Str
		value := []byte(ctx.Args[2].Str)

		ctx.Store.Set(key, value)
		return ctx.Writer.WriteSimpleString("OK")
	}

	return ctx.Writer.WriteError("ERR wrong number of arguments for 'set' command")
}

func cmdGet(ctx *CommandContext) error {
	if len(ctx.Args) == 2 {
		key := ctx.Args[1].Str

		value, ok := ctx.Store.Get(key)
		if !ok {
			return ctx.Writer.WriteNull()
		}

		return ctx.Writer.WriteBulkString(string(value))
	}

	return ctx.Writer.WriteError("ERR wrong number of arguments for 'get' command")
}

func cmdDel(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'del' command")
	}

	keys := make([]string, len(ctx.Args)-1)
	for i := 1; i < len(ctx.Args); i++ {
		keys[i-1] = ctx.Args[i].Str
	}

	delCount := ctx.Store.Del(keys...)
	return ctx.Writer.WriteInteger(int64(delCount))
}

func cmdExists(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'exists' command")
	}

	keys := make([]string, len(ctx.Args)-1)
	for i := 1; i < len(ctx.Args); i++ {
		keys[i-1] = ctx.Args[i].Str
	}

	existsCount := ctx.Store.Exists(keys...)
	return ctx.Writer.WriteInteger(int64(existsCount))
}

func cmdIncr(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'incr' command")
	}

	key := ctx.Args[1].Str

	n, err := ctx.Store.Incr(key)
	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(n)
}

func cmdIncrBy(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'incrBy' command")
	}

	key := ctx.Args[1].Str
	num, err := strconv.ParseInt(ctx.Args[2].Str, 10, 64)
	if err != nil {
		return ctx.Writer.WriteError("ERR value is not an integer or out of range")
	}

	n, err := ctx.Store.IncrBy(key, num)
	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(n)
}

func cmdDecr(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'decr' command")
	}

	key := ctx.Args[1].Str

	n, err := ctx.Store.Decr(key)
	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(n)
}

func cmdDecrBy(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'decrBy' command")
	}

	key := ctx.Args[1].Str
	num, err := strconv.ParseInt(ctx.Args[2].Str, 10, 64)
	if err != nil {
		return ctx.Writer.WriteError("ERR value is not an integer or out of range")
	}

	n, err := ctx.Store.DecrBy(key, num)
	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(n)
}

func cmdMSet(ctx *CommandContext) error {
	if len(ctx.Args) < 3 || (len(ctx.Args)-1)%2 != 0 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'mset' command")
	}

	pairs := make(map[string][]byte)

	for i := 1; i < len(ctx.Args); i += 2 {
		pairs[ctx.Args[i].Str] = []byte(ctx.Args[i+1].Str)
	}

	ctx.Store.MSet(pairs)
	return ctx.Writer.WriteSimpleString("OK")
}

func cmdMGet(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'mget' command")
	}

	keys := make([]string, len(ctx.Args)-1)
	for i, v := range ctx.Args[1:] {
		keys[i] = v.Str
	}

	values := ctx.Store.MGet(keys)
	resp := make([]protocol.RESPValue, len(values))
	for i, v := range values {
		if v != nil {
			resp[i] = protocol.RESPValue{
				Type: protocol.BulkString,
				Str:  string(v),
			}
			continue
		}

		resp[i] = protocol.RESPValue{
			Type:   protocol.BulkString,
			IsNull: true,
		}
	}

	return ctx.Writer.WriteArray(resp)
}

func cmdQuit(ctx *CommandContext) error {
	ctx.Writer.WriteSimpleString("OK")
	ctx.Writer.Flush()
	return ErrQuit
}
