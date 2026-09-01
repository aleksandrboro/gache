package storage

import "github.com/aleksandrboro/gache/internal/datastruct"

const (
	stringType = "string"
	listType   = "list"
	hashType   = "hash"
	setType    = "set"
	zSetType   = "zset"
)

type Value interface {
	Type() string
}

type StringValue struct {
	Data []byte
}

func (v StringValue) Type() string {
	return stringType
}

type ListValue struct {
	Data [][]byte
}

func (v ListValue) Type() string {
	return listType
}

type HashValue struct {
	Data map[string][]byte
}

func (v HashValue) Type() string {
	return hashType
}

type SetValue struct {
	Data map[string]struct{}
}

func (v SetValue) Type() string {
	return setType
}

type ZSetValue struct {
	Data *datastruct.ZSet
}

func (v ZSetValue) Type() string {
	return zSetType
}
