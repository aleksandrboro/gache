package command

import (
	"errors"
	"strconv"
	"time"

	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/storage"
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

func cmdExpire(ctx *CommandContext) error {
	if len(ctx.Args) != 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'expire' command")
	}

	seconds, err := strconv.Atoi(ctx.Args[2].Str)
	if err != nil {
		return ctx.Writer.WriteError("ERR value is not an integer")
	}

	ok := ctx.Store.Expire(ctx.Args[1].Str, (time.Duration(seconds) * time.Second).Nanoseconds())
	if ok {
		return ctx.Writer.WriteInteger(1)
	}

	return ctx.Writer.WriteInteger(0)
}

func cmdPExpire(ctx *CommandContext) error {
	if len(ctx.Args) != 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'pexpire' command")
	}

	millisecs, err := strconv.Atoi(ctx.Args[2].Str)
	if err != nil {
		return ctx.Writer.WriteError("ERR value is not an integer")
	}

	ok := ctx.Store.Expire(ctx.Args[1].Str, (time.Duration(millisecs) * time.Millisecond).Nanoseconds())
	if ok {
		return ctx.Writer.WriteInteger(1)
	}

	return ctx.Writer.WriteInteger(0)
}

func cmdTTL(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'ttl' command")
	}

	nanosecs := ctx.Store.TTL(ctx.Args[1].Str)

	if nanosecs == -1 || nanosecs == -2 {
		return ctx.Writer.WriteInteger(nanosecs)
	}

	return ctx.Writer.WriteInteger(nanosecs / int64(time.Second))
}

func cmdPTTL(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'pttl' command")
	}

	nanosecs := ctx.Store.TTL(ctx.Args[1].Str)

	if nanosecs == -1 || nanosecs == -2 {
		return ctx.Writer.WriteInteger(nanosecs)
	}

	return ctx.Writer.WriteInteger(nanosecs / int64(time.Millisecond))
}

func cmdPersist(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'persist' command")
	}

	ok := ctx.Store.Persist(ctx.Args[1].Str)
	if ok {
		return ctx.Writer.WriteInteger(1)
	}

	return ctx.Writer.WriteInteger(0)
}

func cmdLPush(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'lpush' command")
	}

	key := ctx.Args[1]
	values := make([][]byte, 0, len(ctx.Args[2:]))

	for _, v := range ctx.Args[2:] {
		values = append(values, []byte(v.Str))
	}

	num, err := ctx.Store.LPush(key.Str, values...)
	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(num))
}

func cmdRPush(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'rpush' command")
	}

	key := ctx.Args[1]
	values := make([][]byte, 0, len(ctx.Args[2:]))

	for _, v := range ctx.Args[2:] {
		values = append(values, []byte(v.Str))
	}

	num, err := ctx.Store.RPush(key.Str, values...)
	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(num))
}

func cmdLPop(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'lpop' command")
	}

	key := ctx.Args[1]

	value, err := ctx.Store.LPop(key.Str)
	if err != nil {
		if errors.Is(err, storage.ErrEmptyList) {
			return ctx.Writer.WriteNull()
		}
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteBulkString(string(value))
}

func cmdRPop(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'rpop' command")
	}

	key := ctx.Args[1]

	value, err := ctx.Store.RPop(key.Str)
	if err != nil {
		if errors.Is(err, storage.ErrEmptyList) {
			return ctx.Writer.WriteNull()
		}
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteBulkString(string(value))
}

func cmdLLen(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'llen' command")
	}

	key := ctx.Args[1]

	num, err := ctx.Store.LLen(key.Str)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(num))
}

func cmdLRange(ctx *CommandContext) error {
	if len(ctx.Args) != 4 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'lrange' command")
	}

	key := ctx.Args[1]

	start, err := strconv.ParseInt(ctx.Args[2].Str, 10, 64)
	if err != nil {
		return ctx.Writer.WriteError("ERR value is not an integer")
	}

	stop, err := strconv.ParseInt(ctx.Args[3].Str, 10, 64)
	if err != nil {
		return ctx.Writer.WriteError("ERR value is not an integer")
	}

	list, err := ctx.Store.LRange(key.Str, int(start), int(stop))
	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	resp := make([]protocol.RESPValue, len(list))

	for i, v := range list {
		resp[i] = protocol.RESPValue{
			Type: protocol.BulkString,
			Str:  string(v),
		}
	}

	return ctx.Writer.WriteArray(resp)
}

func cmdHSet(ctx *CommandContext) error {
	if len(ctx.Args) != 4 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'hset' command")
	}

	key := ctx.Args[1].Str
	field := ctx.Args[2].Str
	value := []byte(ctx.Args[3].Str)

	num, err := ctx.Store.HSet(key, field, value)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(num))
}

func cmdHGet(ctx *CommandContext) error {
	if len(ctx.Args) != 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'hget' command")
	}

	key := ctx.Args[1].Str
	field := ctx.Args[2].Str

	value, err := ctx.Store.HGet(key, field)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	if value == nil {
		return ctx.Writer.WriteNull()
	}

	return ctx.Writer.WriteBulkString(string(value))
}

func cmdHDel(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'hdel' command")
	}

	key := ctx.Args[1].Str
	fields := make([]string, 0, len(ctx.Args[2:]))

	for _, v := range ctx.Args[2:] {
		fields = append(fields, v.Str)
	}

	num, err := ctx.Store.HDel(key, fields...)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(num))
}

func cmdHGetAll(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'hgetall' command")
	}

	key := ctx.Args[1].Str

	fieldsMap, err := ctx.Store.HGetAll(key)
	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	fieldsSlice := make([]protocol.RESPValue, 0, len(fieldsMap))

	for k, v := range fieldsMap {
		fieldsSlice = append(fieldsSlice, protocol.RESPValue{
			Str:  k,
			Type: protocol.BulkString,
		}, protocol.RESPValue{
			Str:  string(v),
			Type: protocol.BulkString,
		})
	}

	return ctx.Writer.WriteArray(fieldsSlice)
}

func cmdHKeys(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'hkeys' command")
	}

	key := ctx.Args[1].Str

	keys, err := ctx.Store.HKeys(key)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	resp := make([]protocol.RESPValue, 0, len(keys))

	for _, k := range keys {
		resp = append(resp, protocol.RESPValue{
			Str:  k,
			Type: protocol.BulkString,
		})
	}

	return ctx.Writer.WriteArray(resp)
}

func cmdHVals(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'hvals' command")
	}

	key := ctx.Args[1].Str

	vals, err := ctx.Store.HVals(key)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	resp := make([]protocol.RESPValue, 0, len(vals))

	for _, v := range vals {
		resp = append(resp, protocol.RESPValue{
			Str:  string(v),
			Type: protocol.BulkString,
		})
	}

	return ctx.Writer.WriteArray(resp)
}

func cmdHLen(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'hlen' command")
	}

	key := ctx.Args[1].Str

	len, err := ctx.Store.HLen(key)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(len))
}

func cmdHExists(ctx *CommandContext) error {
	if len(ctx.Args) != 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'hexists' command")
	}

	key := ctx.Args[1].Str
	field := ctx.Args[2].Str

	exists, err := ctx.Store.HExists(key, field)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	if !exists {
		return ctx.Writer.WriteInteger(0)
	}

	return ctx.Writer.WriteInteger(1)
}

func cmdSAdd(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'sadd' command")
	}

	key := ctx.Args[1].Str
	members := make([]string, 0, len(ctx.Args[2:]))

	for _, arg := range ctx.Args[2:] {
		members = append(members, arg.Str)
	}

	num, err := ctx.Store.SAdd(key, members...)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(num))
}

func cmdSRem(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'srem' command")
	}

	key := ctx.Args[1].Str
	members := make([]string, 0, len(ctx.Args[2:]))

	for _, arg := range ctx.Args[2:] {
		members = append(members, arg.Str)
	}

	num, err := ctx.Store.SRem(key, members...)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(num))
}

func cmdSIsMember(ctx *CommandContext) error {
	if len(ctx.Args) != 3 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'sismember' command")
	}

	key := ctx.Args[1].Str
	member := ctx.Args[2].Str

	isMember, err := ctx.Store.SIsMember(key, member)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	if !isMember {
		return ctx.Writer.WriteInteger(0)
	}

	return ctx.Writer.WriteInteger(1)
}

func cmdSMembers(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'smembers' command")
	}

	key := ctx.Args[1].Str

	members, err := ctx.Store.SMembers(key)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	resp := make([]protocol.RESPValue, 0, len(members))

	for _, member := range members {
		resp = append(resp, protocol.RESPValue{
			Str:  member,
			Type: protocol.BulkString,
		})
	}

	return ctx.Writer.WriteArray(resp)
}

func cmdSCard(ctx *CommandContext) error {
	if len(ctx.Args) != 2 {
		return ctx.Writer.WriteError("ERR wrong number of arguments for 'scard' command")
	}

	key := ctx.Args[1].Str

	num, err := ctx.Store.SCard(key)

	if err != nil {
		return ctx.Writer.WriteError(err.Error())
	}

	return ctx.Writer.WriteInteger(int64(num))
}

func cmdQuit(ctx *CommandContext) error {
	ctx.Writer.WriteSimpleString("OK")
	ctx.Writer.Flush()
	return ErrQuit
}
