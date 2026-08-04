package protocol

type RESPType int

const (
	SimpleString RESPType = iota
	Error
	Integer
	BulkString
	Array
)

type RESPValue struct {
	Type   RESPType
	Str    string
	Int    int64
	Array  []RESPValue
	IsNull bool
}
