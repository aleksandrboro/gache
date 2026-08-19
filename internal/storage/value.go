package storage

const (
	stringType = "string"
	listType   = "list"
	hashType   = "hash"
	setType    = "set"
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
