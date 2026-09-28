package command

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/stretchr/testify/require"
)

// String commands

func TestPing(t *testing.T) {
	router := NewRouter()
	router.Register("PING", cmdPing)

	testCases := []struct {
		Name string
		Ctx  CommandContext
		Exp  string
	}{
		{
			Name: "Upper case PING test",
			Ctx:  CommandContext{Args: []protocol.RESPValue{{Str: "PING"}}},
			Exp:  "+PONG\r\n",
		},
		{
			Name: "Lower case PING test",
			Ctx:  CommandContext{Args: []protocol.RESPValue{{Str: "ping"}}},
			Exp:  "+PONG\r\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buf := bytes.Buffer{}
			w := bufio.NewWriter(&buf)

			tc.Ctx.Writer = protocol.NewWriter(w)

			router.Handle(&tc.Ctx)
			tc.Ctx.Writer.Flush()

			require.Equal(t, tc.Exp, buf.String())
		})
	}
}

func TestEcho(t *testing.T) {
	router := NewRouter()
	router.Register("ECHO", cmdEcho)

	testCases := []struct {
		Name    string
		Ctx     CommandContext
		Exp     string
		IsValid bool
	}{
		{
			Name:    "Upper case ECHO test",
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "ECHO"}, {Str: "hello"}}},
			Exp:     "$5\r\nhello\r\n",
			IsValid: true,
		},
		{
			Name:    "Lower case ECHO test",
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "echo"}, {Str: "hello"}}},
			Exp:     "$5\r\nhello\r\n",
			IsValid: true,
		},
		{
			Name:    "Empty ECHO",
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "ECHO"}}},
			IsValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buf := bytes.Buffer{}
			w := bufio.NewWriter(&buf)

			tc.Ctx.Writer = protocol.NewWriter(w)

			router.Handle(&tc.Ctx)
			tc.Ctx.Writer.Flush()

			if tc.IsValid {
				require.Equal(t, tc.Exp, buf.String())
			} else {
				require.Contains(t, buf.String(), "ERR")
			}
		})
	}
}

func TestSet(t *testing.T) {

}

func TestGet(t *testing.T) {

}

func TestDel(t *testing.T) {

}

func TestExists(t *testing.T) {

}

func TestIncr(t *testing.T) {

}

func TestDecr(t *testing.T) {

}

func TestMSet(t *testing.T) {

}

func TestMGet(t *testing.T) {

}

func TestExpire(t *testing.T) {

}

func TestTTL(t *testing.T) {

}

func TestPersists(t *testing.T) {

}
