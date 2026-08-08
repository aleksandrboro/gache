package protocol

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_WriteSimpleString(t *testing.T) {
	resp := RESPValue{
		Type: SimpleString,
		Str:  "OK",
	}

	var buf bytes.Buffer
	writer := NewWriter(bufio.NewWriter(&buf))
	err := writer.WriteSimpleString(resp.Str)
	require.NoError(t, err)
	writer.Flush()

	reader := bufio.NewReader(&buf)
	parser := NewParser(reader)
	parsed, err := parser.Parse()
	require.NoError(t, err)

	require.Equal(t, resp.Type, parsed.Type)
	require.Equal(t, resp.Str, parsed.Str)
}

func Test_WriteError(t *testing.T) {
	resp := RESPValue{
		Type: Error,
		Str:  "error",
	}

	var buf bytes.Buffer
	writer := NewWriter(bufio.NewWriter(&buf))
	err := writer.WriteError(resp.Str)
	require.NoError(t, err)
	writer.Flush()

	reader := bufio.NewReader(&buf)
	parser := NewParser(reader)
	parsed, err := parser.Parse()
	require.NoError(t, err)

	require.Equal(t, resp.Type, parsed.Type)
	require.Equal(t, resp.Str, parsed.Str)
}

func Test_WriteInteger(t *testing.T) {
	resp := RESPValue{
		Type: Integer,
		Int:  2,
	}

	var buf bytes.Buffer
	writer := NewWriter(bufio.NewWriter(&buf))
	err := writer.WriteInteger(resp.Int)
	require.NoError(t, err)
	writer.Flush()

	reader := bufio.NewReader(&buf)
	parser := NewParser(reader)
	parsed, err := parser.Parse()
	require.NoError(t, err)

	require.Equal(t, resp.Type, parsed.Type)
	require.Equal(t, resp.Int, parsed.Int)
}

func Test_WriteBulkString(t *testing.T) {
	resp := RESPValue{
		Type: BulkString,
		Str:  "Hello world",
	}

	var buf bytes.Buffer
	writer := NewWriter(bufio.NewWriter(&buf))
	err := writer.WriteBulkString(resp.Str)
	require.NoError(t, err)
	writer.Flush()

	reader := bufio.NewReader(&buf)
	parser := NewParser(reader)
	parsed, err := parser.Parse()
	require.NoError(t, err)

	require.Equal(t, resp.Type, parsed.Type)
	require.Equal(t, resp.Str, parsed.Str)
}

func Test_WriteNull(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(bufio.NewWriter(&buf))
	err := writer.WriteNull()
	require.NoError(t, err)
	writer.Flush()

	reader := bufio.NewReader(&buf)
	parser := NewParser(reader)
	parsed, err := parser.Parse()
	require.NoError(t, err)

	require.True(t, parsed.IsNull)
}

func Test_WriteArray(t *testing.T) {
	resp := RESPValue{
		Type: Array,
		Array: []RESPValue{
			{
				Type: BulkString,
				Str:  "SET",
			},
			{
				Type: BulkString,
				Str:  "key",
			},
			{
				Type: BulkString,
				Str:  "value",
			},
		},
	}

	var buf bytes.Buffer
	writer := NewWriter(bufio.NewWriter(&buf))
	err := writer.WriteArray(resp.Array)
	require.NoError(t, err)
	writer.Flush()

	reader := bufio.NewReader(&buf)
	parser := NewParser(reader)
	parsed, err := parser.Parse()
	require.NoError(t, err)

	require.Equal(t, resp.Type, parsed.Type)
	for i := range parsed.Array {
		require.Equal(t, resp.Array[i].Str, parsed.Array[i].Str)
		require.Equal(t, resp.Array[i].Type, parsed.Array[i].Type)
	}
}
