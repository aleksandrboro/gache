package aof

import (
	"bufio"
	"os"
	"strconv"
	"time"

	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/storage"
)

func Rewrite(filename string, store *storage.Store) error {
	file, err := os.OpenFile(filename+".tmp", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	writer := protocol.NewWriter(bufio.NewWriter(file))

	store.ForEach(func(key string, entry *storage.Entry) {
		if entry.ExpireAt > 0 && time.Now().UnixNano() > entry.ExpireAt {
			return
		}

		switch entry.Value.Type() {
		case "string":
			writer.WriteArray([]protocol.RESPValue{{Str: "SET", Type: protocol.BulkString}, {Str: key, Type: protocol.BulkString}, {Str: string(entry.Value.(storage.StringValue).Data), Type: protocol.BulkString}})
		case "list":
			array := make([]protocol.RESPValue, 0, 2+len(entry.Value.(storage.ListValue).Data))
			array = append(array, protocol.RESPValue{Str: "RPUSH", Type: protocol.BulkString}, protocol.RESPValue{Str: key, Type: protocol.BulkString})

			for _, v := range entry.Value.(storage.ListValue).Data {
				array = append(array, protocol.RESPValue{Str: string(v), Type: protocol.BulkString})
			}

			writer.WriteArray(array)
		case "hash":
			array := make([]protocol.RESPValue, 0, 2+len(entry.Value.(storage.HashValue).Data)*2)
			array = append(array, protocol.RESPValue{Str: "HSET", Type: protocol.BulkString}, protocol.RESPValue{Str: key, Type: protocol.BulkString})

			for k, v := range entry.Value.(storage.HashValue).Data {
				array = append(array, protocol.RESPValue{Str: k, Type: protocol.BulkString}, protocol.RESPValue{Str: string(v), Type: protocol.BulkString})
			}

			writer.WriteArray(array)
		case "set":
			array := make([]protocol.RESPValue, 0, 2+len(entry.Value.(storage.SetValue).Data))
			array = append(array, protocol.RESPValue{Str: "SADD", Type: protocol.BulkString}, protocol.RESPValue{Str: key, Type: protocol.BulkString})

			for v := range entry.Value.(storage.SetValue).Data {
				array = append(array, protocol.RESPValue{Str: v, Type: protocol.BulkString})
			}

			writer.WriteArray(array)
		case "zset":
			array := make([]protocol.RESPValue, 0, 2+entry.Value.(storage.ZSetValue).Data.Len()*2)
			array = append(array, protocol.RESPValue{Str: "ZADD", Type: protocol.BulkString}, protocol.RESPValue{Str: key, Type: protocol.BulkString})

			set := entry.Value.(storage.ZSetValue).Data.Range(0, -1)

			for _, el := range set {
				array = append(array, protocol.RESPValue{Str: strconv.FormatFloat(el.Score, 'f', -1, 64), Type: protocol.BulkString}, protocol.RESPValue{Str: el.Member, Type: protocol.BulkString})
			}

			writer.WriteArray(array)
		}

		if entry.ExpireAt > 0 {
			remainingMs := (entry.ExpireAt - time.Now().UnixNano()) / 1e6
			pexpireArgs := []protocol.RESPValue{
				{Type: protocol.BulkString, Str: "PEXPIRE"},
				{Type: protocol.BulkString, Str: key},
				{Type: protocol.BulkString, Str: strconv.FormatInt(remainingMs, 10)},
			}
			writer.WriteArray(pexpireArgs)
		}
	})

	if err := writer.Flush(); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(filename+".tmp", filename); err != nil {
		return err
	}

	return nil
}
