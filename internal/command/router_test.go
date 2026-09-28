package command

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	router := NewRouter()

	router.Register("PING", cmdPing)
	router.Register("SET", cmdSet)

	testCases := []struct {
		Name    string
		Ctx     CommandContext
		Exp     string
		IsValid bool
	}{
		{
			Name: "PING test",
			Ctx: CommandContext{
				Args: []protocol.RESPValue{{Str: "PING"}},
			},
			Exp:     "+PONG\r\n",
			IsValid: true,
		},
		{
			Name: "SET without args",
			Ctx: CommandContext{
				Args: []protocol.RESPValue{{Str: "set"}},
			},
			IsValid: false,
		},
		{
			Name: "Unknown command",
			Ctx: CommandContext{
				Args: []protocol.RESPValue{{Str: "UNKNOWN"}},
			},
			IsValid: false,
		},
		{
			Name:    "Empty command",
			Ctx:     CommandContext{},
			IsValid: false,
		},
		{
			Name: "Set test",
			Ctx: CommandContext{
				Args: []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "value"}},
			},
			Exp:     "+OK\r\n",
			IsValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buf := bytes.Buffer{}
			w := bufio.NewWriter(&buf)

			tc.Ctx.Writer = protocol.NewWriter(w)
			tc.Ctx.Store = storage.NewStore()

			router.Handle(&tc.Ctx)
			tc.Ctx.Writer.Flush()

			if tc.IsValid {
				if len(tc.Ctx.Args) > 2 {
					v, ok := tc.Ctx.Store.Get(tc.Ctx.Args[1].Str)
					require.True(t, ok)
					require.Equal(t, string(v), tc.Ctx.Args[2].Str)
				}

				require.Equal(t, tc.Exp, buf.String())
			} else {
				require.Contains(t, buf.String(), "ERR")
			}
		})
	}
}
