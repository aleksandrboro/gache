package command

import (
	"bufio"
	"bytes"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aleksandrboro/gache/internal/protocol"
	"github.com/aleksandrboro/gache/internal/storage"
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
	router := NewRouter()
	router.Register("SET", cmdSet)

	testCases := []struct {
		Name    string
		Ctx     CommandContext
		Exp     string
		IsValid bool
	}{
		{
			Name:    "Correct SET",
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "value"}}},
			Exp:     "+OK\r\n",
			IsValid: true,
		},
		{
			Name:    "Empty SET",
			Ctx:     CommandContext{Args: []protocol.RESPValue{}},
			IsValid: false,
		},
		{
			Name:    "SET without value",
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "SET"}, {Str: "key"}}},
			IsValid: false,
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
				v, ok := tc.Ctx.Store.Get(tc.Ctx.Args[1].Str)
				require.True(t, ok)
				require.Equal(t, tc.Ctx.Args[2].Str, string(v))
				require.Equal(t, tc.Exp, buf.String())
			} else {
				require.Contains(t, buf.String(), "ERR")
			}
		})
	}

	t.Run("Rewrite key value", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		stor := storage.NewStore()

		ctx1 := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "value1"}},
			Writer: protocol.NewWriter(w),
			Store:  stor,
		}

		ctx2 := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "value2"}},
			Writer: protocol.NewWriter(w),
			Store:  stor,
		}

		router.Handle(&ctx1)
		ctx1.Writer.Flush()
		require.Equal(t, buf.String(), "+OK\r\n")
		buf.Reset()

		router.Handle(&ctx2)
		ctx2.Writer.Flush()
		require.Equal(t, buf.String(), "+OK\r\n")
		buf.Reset()

		exp := "value2"

		v, ok := ctx1.Store.Get(ctx1.Args[1].Str)

		require.True(t, ok)
		require.Equal(t, exp, string(v))
	})
}

func TestGet(t *testing.T) {
	router := NewRouter()
	router.Register("GET", cmdGet)
	router.Register("SET", cmdSet)

	testCases := []struct {
		Name    string
		Setup   []CommandContext // предварительные команды
		Ctx     CommandContext
		Exp     string
		IsValid bool
	}{
		{
			Name: "Get existing key",
			Setup: []CommandContext{
				{Args: []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "value"}}},
			},
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "GET"}, {Str: "key"}}},
			Exp:     "$5\r\nvalue\r\n",
			IsValid: true,
		},
		{
			Name:    "Get missing key",
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "GET"}, {Str: "missing"}}},
			Exp:     "$-1\r\n",
			IsValid: true,
		},
		{
			Name:    "GET without args",
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "GET"}}},
			IsValid: false,
		},
		{
			Name:    "GET with too many args",
			Ctx:     CommandContext{Args: []protocol.RESPValue{{Str: "GET"}, {Str: "a"}, {Str: "b"}}},
			IsValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buf := bytes.Buffer{}
			w := bufio.NewWriter(&buf)
			store := storage.NewStore()

			// Setup
			for _, setup := range tc.Setup {
				setup.Writer = protocol.NewWriter(w)
				setup.Store = store
				router.Handle(&setup)
				w.Flush()
				buf.Reset()
			}

			// Test
			tc.Ctx.Writer = protocol.NewWriter(w)
			tc.Ctx.Store = store
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

func TestDel(t *testing.T) {
	router := NewRouter()
	router.Register("DEL", cmdDel)
	router.Register("SET", cmdSet)

	t.Run("Del one key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		// Setup: SET a 1, SET b 2
		for _, args := range [][]protocol.RESPValue{
			{{Str: "SET"}, {Str: "a"}, {Str: "1"}},
			{{Str: "SET"}, {Str: "b"}, {Str: "2"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
		}

		// DEL a
		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "DEL"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
		_, ok := store.Get("a")
		require.False(t, ok)
	})

	t.Run("Del multiple keys", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		for _, args := range [][]protocol.RESPValue{
			{{Str: "SET"}, {Str: "a"}, {Str: "1"}},
			{{Str: "SET"}, {Str: "b"}, {Str: "2"}},
			{{Str: "SET"}, {Str: "c"}, {Str: "3"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
		}

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "DEL"}, {Str: "a"}, {Str: "b"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":2\r\n", buf.String())
		_, ok := store.Get("c")
		require.True(t, ok)
	})

	t.Run("Del missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "DEL"}, {Str: "x"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("DEL without args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "DEL"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestExists(t *testing.T) {
	router := NewRouter()
	router.Register("EXISTS", cmdExists)
	router.Register("SET", cmdSet)

	t.Run("Exists existing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "a"}, {Str: "1"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXISTS"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("Exists missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXISTS"}, {Str: "x"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("Exists multiple keys", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		for _, args := range [][]protocol.RESPValue{
			{{Str: "SET"}, {Str: "a"}, {Str: "1"}},
			{{Str: "SET"}, {Str: "b"}, {Str: "2"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
		}

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXISTS"}, {Str: "a"}, {Str: "b"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":2\r\n", buf.String())
	})

	t.Run("Exists with duplicates", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "a"}, {Str: "1"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXISTS"}, {Str: "a"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":2\r\n", buf.String())
	})

	t.Run("EXISTS without args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXISTS"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestIncr(t *testing.T) {
	router := NewRouter()
	router.Register("INCR", cmdIncr)
	router.Register("SET", cmdSet)

	t.Run("Incr existing number", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "c"}, {Str: "10"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "INCR"}, {Str: "c"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":11\r\n", buf.String())

		v, ok := store.Get("c")
		require.True(t, ok)
		require.Equal(t, "11", string(v))
	})

	t.Run("Incr missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "INCR"}, {Str: "c"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())

		v, ok := store.Get("c")
		require.True(t, ok)
		require.Equal(t, "1", string(v))
	})

	t.Run("Incr non-integer value", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "name"}, {Str: "Alice"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "INCR"}, {Str: "name"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})

	t.Run("INCR without args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "INCR"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestDecr(t *testing.T) {
	router := NewRouter()
	router.Register("DECR", cmdDecr)
	router.Register("SET", cmdSet)

	t.Run("Decr existing number", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "c"}, {Str: "10"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "DECR"}, {Str: "c"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":9\r\n", buf.String())
	})

	t.Run("Decr missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "DECR"}, {Str: "c"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":-1\r\n", buf.String())
	})

	t.Run("Decr non-integer value", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "name"}, {Str: "Alice"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "DECR"}, {Str: "name"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestMSet(t *testing.T) {
	router := NewRouter()
	router.Register("MSET", cmdMSet)

	t.Run("MSet 3 pairs", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "MSET"}, {Str: "a"}, {Str: "1"}, {Str: "b"}, {Str: "2"}, {Str: "c"}, {Str: "3"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "+OK\r\n", buf.String())

		for _, pair := range [][2]string{{"a", "1"}, {"b", "2"}, {"c", "3"}} {
			v, ok := store.Get(pair[0])
			require.True(t, ok)
			require.Equal(t, pair[1], string(v))
		}
	})

	t.Run("MSet odd number of args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "MSET"}, {Str: "a"}, {Str: "1"}, {Str: "b"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})

	t.Run("MSET without args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "MSET"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestMGet(t *testing.T) {
	router := NewRouter()
	router.Register("MGET", cmdMGet)
	router.Register("SET", cmdSet)

	t.Run("MGet all existing", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		for _, args := range [][]protocol.RESPValue{
			{{Str: "SET"}, {Str: "a"}, {Str: "1"}},
			{{Str: "SET"}, {Str: "b"}, {Str: "2"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
		}

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "MGET"}, {Str: "a"}, {Str: "b"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*2\r\n$1\r\n1\r\n$1\r\n2\r\n", buf.String())
	})

	t.Run("MGet with missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "a"}, {Str: "1"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "MGET"}, {Str: "a"}, {Str: "x"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*2\r\n$1\r\n1\r\n$-1\r\n", buf.String())
	})

	t.Run("MGET without args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "MGET"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestExpire(t *testing.T) {
	router := NewRouter()
	router.Register("EXPIRE", cmdExpire)
	router.Register("SET", cmdSet)
	router.Register("TTL", cmdTTL)

	t.Run("Expire existing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "val"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXPIRE"}, {Str: "key"}, {Str: "10"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())

		// Check TTL > 0
		buf.Reset()
		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "TTL"}, {Str: "key"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), ":")
		require.NotEqual(t, ":-1\r\n", buf.String())
		require.NotEqual(t, ":-2\r\n", buf.String())
	})

	t.Run("Expire missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXPIRE"}, {Str: "x"}, {Str: "10"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("Expire non-integer seconds", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "val"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXPIRE"}, {Str: "key"}, {Str: "abc"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestTTL(t *testing.T) {
	router := NewRouter()
	router.Register("TTL", cmdTTL)
	router.Register("SET", cmdSet)
	router.Register("EXPIRE", cmdExpire)

	t.Run("TTL with expiry", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		// SET key val
		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "val"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		// EXPIRE key 100
		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXPIRE"}, {Str: "key"}, {Str: "100"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		// TTL key
		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "TTL"}, {Str: "key"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		// Should be a positive integer
		require.Contains(t, buf.String(), ":")
		require.NotContains(t, buf.String(), ":-1")
		require.NotContains(t, buf.String(), ":-2")
	})

	t.Run("TTL without expiry", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "val"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "TTL"}, {Str: "key"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":-1\r\n", buf.String())
	})

	t.Run("TTL missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "TTL"}, {Str: "x"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":-2\r\n", buf.String())
	})
}

func TestPersist(t *testing.T) {
	router := NewRouter()
	router.Register("PERSIST", cmdPersist)
	router.Register("SET", cmdSet)
	router.Register("EXPIRE", cmdExpire)
	router.Register("TTL", cmdTTL)

	t.Run("Persist key with TTL", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		// SET key val
		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "val"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		// EXPIRE key 100
		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "EXPIRE"}, {Str: "key"}, {Str: "100"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		// PERSIST key
		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "PERSIST"}, {Str: "key"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())

		// TTL should be -1 now
		buf.Reset()
		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "TTL"}, {Str: "key"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":-1\r\n", buf.String())
	})

	t.Run("Persist key without TTL", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "key"}, {Str: "val"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "PERSIST"}, {Str: "key"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("Persist missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "PERSIST"}, {Str: "x"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

// ==================== List commands ====================

func TestLPush(t *testing.T) {
	router := NewRouter()
	router.Register("LPUSH", cmdLPush)
	router.Register("LLEN", cmdLLen)
	router.Register("SET", cmdSet)

	t.Run("LPush to new list", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "LPUSH"}, {Str: "mylist"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("LPush multiple values", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "LPUSH"}, {Str: "mylist"}, {Str: "a"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":3\r\n", buf.String())
	})

	t.Run("LPush to existing list", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		// LPUSH mylist a
		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "LPUSH"}, {Str: "mylist"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		// LPUSH mylist b
		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "LPUSH"}, {Str: "mylist"}, {Str: "b"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":2\r\n", buf.String())
	})

	t.Run("LPUSH without enough args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "LPUSH"}, {Str: "mylist"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})

	t.Run("LPUSH on string key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SET"}, {Str: "mystr"}, {Str: "hello"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "LPUSH"}, {Str: "mystr"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "WRONGTYPE")
	})
}

func TestRPush(t *testing.T) {
	router := NewRouter()
	router.Register("RPUSH", cmdRPush)

	t.Run("RPush to new list", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "RPUSH"}, {Str: "mylist"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("RPush multiple values", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "RPUSH"}, {Str: "mylist"}, {Str: "a"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":3\r\n", buf.String())
	})

	t.Run("RPUSH without enough args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "RPUSH"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestLPop(t *testing.T) {
	router := NewRouter()
	router.Register("LPOP", cmdLPop)
	router.Register("RPUSH", cmdRPush)

	t.Run("LPop from list", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		// RPUSH mylist a b c
		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "RPUSH"}, {Str: "mylist"}, {Str: "a"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		// LPOP mylist
		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "LPOP"}, {Str: "mylist"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$1\r\na\r\n", buf.String())
	})

	t.Run("LPop from empty list", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "LPOP"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$-1\r\n", buf.String())
	})
}

func TestRPop(t *testing.T) {
	router := NewRouter()
	router.Register("RPOP", cmdRPop)
	router.Register("RPUSH", cmdRPush)

	t.Run("RPop from list", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "RPUSH"}, {Str: "mylist"}, {Str: "a"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "RPOP"}, {Str: "mylist"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$1\r\nc\r\n", buf.String())
	})

	t.Run("RPop from missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "RPOP"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$-1\r\n", buf.String())
	})
}

func TestLLen(t *testing.T) {
	router := NewRouter()
	router.Register("LLEN", cmdLLen)
	router.Register("RPUSH", cmdRPush)

	t.Run("LLen of existing list", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "RPUSH"}, {Str: "mylist"}, {Str: "a"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "LLEN"}, {Str: "mylist"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":3\r\n", buf.String())
	})

	t.Run("LLen of missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "LLEN"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

func TestLRange(t *testing.T) {
	router := NewRouter()
	router.Register("LRANGE", cmdLRange)
	router.Register("RPUSH", cmdRPush)

	t.Run("LRange all elements", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "RPUSH"}, {Str: "mylist"}, {Str: "a"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args: []protocol.RESPValue{
				{Str: "LRANGE"}, {Str: "mylist"}, {Str: "0"}, {Str: "-1"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*3\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n", buf.String())
	})

	t.Run("LRange subset", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "RPUSH"}, {Str: "mylist"}, {Str: "a"}, {Str: "b"}, {Str: "c"}, {Str: "d"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args: []protocol.RESPValue{
				{Str: "LRANGE"}, {Str: "mylist"}, {Str: "1"}, {Str: "2"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*2\r\n$1\r\nb\r\n$1\r\nc\r\n", buf.String())
	})

	t.Run("LRange missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "LRANGE"}, {Str: "missing"}, {Str: "0"}, {Str: "-1"},
			},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*0\r\n", buf.String())
	})
}

// ==================== Hash commands ====================

func TestHSet(t *testing.T) {
	router := NewRouter()
	router.Register("HSET", cmdHSet)

	t.Run("HSet new field", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("HSet overwrite field", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Bob"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("HSET without enough args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HSET"}, {Str: "user"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestHGet(t *testing.T) {
	router := NewRouter()
	router.Register("HGET", cmdHGet)
	router.Register("HSET", cmdHSet)

	t.Run("HGet existing field", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "HGET"}, {Str: "user"}, {Str: "name"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$5\r\nAlice\r\n", buf.String())
	})

	t.Run("HGet missing field", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "HGET"}, {Str: "user"}, {Str: "age"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$-1\r\n", buf.String())
	})

	t.Run("HGet missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HGET"}, {Str: "missing"}, {Str: "name"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$-1\r\n", buf.String())
	})
}

func TestHDel(t *testing.T) {
	router := NewRouter()
	router.Register("HDEL", cmdHDel)
	router.Register("HSET", cmdHSet)

	t.Run("HDel existing field", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		for _, args := range [][]protocol.RESPValue{
			{{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"}},
			{{Str: "HSET"}, {Str: "user"}, {Str: "age"}, {Str: "30"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
		}

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HDEL"}, {Str: "user"}, {Str: "name"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("HDel missing field", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "HDEL"}, {Str: "user"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

func TestHGetAll(t *testing.T) {
	router := NewRouter()
	router.Register("HGETALL", cmdHGetAll)
	router.Register("HSET", cmdHSet)

	t.Run("HGetAll with fields", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "HGETALL"}, {Str: "user"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		// Should contain name and Alice (order may vary due to map)
		result := buf.String()
		require.Contains(t, result, "name")
		require.Contains(t, result, "Alice")
		require.Contains(t, result, "*2\r\n")
	})

	t.Run("HGetAll missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HGETALL"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*0\r\n", buf.String())
	})
}

func TestHKeys(t *testing.T) {
	router := NewRouter()
	router.Register("HKEYS", cmdHKeys)
	router.Register("HSET", cmdHSet)

	t.Run("HKeys with fields", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "HKEYS"}, {Str: "user"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "name")
	})

	t.Run("HKeys missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HKEYS"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*0\r\n", buf.String())
	})
}

func TestHVals(t *testing.T) {
	router := NewRouter()
	router.Register("HVALS", cmdHVals)
	router.Register("HSET", cmdHSet)

	t.Run("HVals with fields", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "HVALS"}, {Str: "user"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "Alice")
	})

	t.Run("HVals missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HVALS"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*0\r\n", buf.String())
	})
}

func TestHLen(t *testing.T) {
	router := NewRouter()
	router.Register("HLEN", cmdHLen)
	router.Register("HSET", cmdHSet)

	t.Run("HLen with fields", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		for _, args := range [][]protocol.RESPValue{
			{{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"}},
			{{Str: "HSET"}, {Str: "user"}, {Str: "age"}, {Str: "30"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
		}

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HLEN"}, {Str: "user"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":2\r\n", buf.String())
	})

	t.Run("HLen missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HLEN"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

func TestHExists(t *testing.T) {
	router := NewRouter()
	router.Register("HEXISTS", cmdHExists)
	router.Register("HSET", cmdHSet)

	t.Run("HExists existing field", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "HEXISTS"}, {Str: "user"}, {Str: "name"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("HExists missing field", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "HSET"}, {Str: "user"}, {Str: "name"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "HEXISTS"}, {Str: "user"}, {Str: "age"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("HExists missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "HEXISTS"}, {Str: "missing"}, {Str: "name"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

// ==================== Set commands ====================

func TestSAdd(t *testing.T) {
	router := NewRouter()
	router.Register("SADD", cmdSAdd)

	t.Run("SAdd new members", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "a"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":3\r\n", buf.String())
	})

	t.Run("SAdd with duplicates", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "a"}, {Str: "b"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("SADD without enough args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SADD"}, {Str: "myset"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestSRem(t *testing.T) {
	router := NewRouter()
	router.Register("SREM", cmdSRem)
	router.Register("SADD", cmdSAdd)

	t.Run("SRem existing member", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "a"}, {Str: "b"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "SREM"}, {Str: "myset"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("SRem missing member", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "a"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "SREM"}, {Str: "myset"}, {Str: "x"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("SRem missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SREM"}, {Str: "missing"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

func TestSIsMember(t *testing.T) {
	router := NewRouter()
	router.Register("SISMEMBER", cmdSIsMember)
	router.Register("SADD", cmdSAdd)

	t.Run("SIsMember existing", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "a"}, {Str: "b"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "SISMEMBER"}, {Str: "myset"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("SIsMember missing", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "a"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "SISMEMBER"}, {Str: "myset"}, {Str: "x"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("SIsMember missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SISMEMBER"}, {Str: "missing"}, {Str: "a"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

func TestSMembers(t *testing.T) {
	router := NewRouter()
	router.Register("SMEMBERS", cmdSMembers)
	router.Register("SADD", cmdSAdd)

	t.Run("SMembers with elements", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "a"}, {Str: "b"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "SMEMBERS"}, {Str: "myset"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		result := buf.String()
		require.Contains(t, result, "*2\r\n")
		require.Contains(t, result, "a")
		require.Contains(t, result, "b")
	})

	t.Run("SMembers missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SMEMBERS"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*0\r\n", buf.String())
	})
}

func TestSCard(t *testing.T) {
	router := NewRouter()
	router.Register("SCARD", cmdSCard)
	router.Register("SADD", cmdSAdd)

	t.Run("SCard with elements", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "SADD"}, {Str: "myset"}, {Str: "a"}, {Str: "b"}, {Str: "c"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "SCARD"}, {Str: "myset"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":3\r\n", buf.String())
	})

	t.Run("SCard missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "SCARD"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

// ==================== Sorted Set commands ====================

func TestZAdd(t *testing.T) {
	router := NewRouter()
	router.Register("ZADD", cmdZAdd)

	t.Run("ZAdd new member", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("ZAdd update score", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZADD"}, {Str: "lb"}, {Str: "200"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("ZAdd invalid score", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZADD"}, {Str: "lb"}, {Str: "abc"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})

	t.Run("ZADD without enough args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZADD"}, {Str: "lb"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "ERR")
	})
}

func TestZRem(t *testing.T) {
	router := NewRouter()
	router.Register("ZREM", cmdZRem)
	router.Register("ZADD", cmdZAdd)

	t.Run("ZRem existing member", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZREM"}, {Str: "lb"}, {Str: "Alice"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":1\r\n", buf.String())
	})

	t.Run("ZRem missing member", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZREM"}, {Str: "lb"}, {Str: "Bob"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})

	t.Run("ZRem missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZREM"}, {Str: "missing"}, {Str: "Alice"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

func TestZScore(t *testing.T) {
	router := NewRouter()
	router.Register("ZSCORE", cmdZScore)
	router.Register("ZADD", cmdZAdd)

	t.Run("ZScore existing member", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZSCORE"}, {Str: "lb"}, {Str: "Alice"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$3\r\n100\r\n", buf.String())
	})

	t.Run("ZScore missing member", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		w.Flush()
		buf.Reset()

		ctx = CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZSCORE"}, {Str: "lb"}, {Str: "Bob"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$-1\r\n", buf.String())
	})

	t.Run("ZScore missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZSCORE"}, {Str: "missing"}, {Str: "Alice"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "$-1\r\n", buf.String())
	})
}

func TestZCard(t *testing.T) {
	router := NewRouter()
	router.Register("ZCARD", cmdZCard)
	router.Register("ZADD", cmdZAdd)

	t.Run("ZCard with elements", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		for _, args := range [][]protocol.RESPValue{
			{{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"}},
			{{Str: "ZADD"}, {Str: "lb"}, {Str: "200"}, {Str: "Bob"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
		}

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZCARD"}, {Str: "lb"}},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":2\r\n", buf.String())
	})

	t.Run("ZCard missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args:   []protocol.RESPValue{{Str: "ZCARD"}, {Str: "missing"}},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, ":0\r\n", buf.String())
	})
}

func TestZRange(t *testing.T) {
	router := NewRouter()
	router.Register("ZRANGE", cmdZRange)
	router.Register("ZADD", cmdZAdd)

	t.Run("ZRange all elements", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		for _, args := range [][]protocol.RESPValue{
			{{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"}},
			{{Str: "ZADD"}, {Str: "lb"}, {Str: "200"}, {Str: "Bob"}},
			{{Str: "ZADD"}, {Str: "lb"}, {Str: "150"}, {Str: "Charlie"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
			w.Reset(&buf)
		}

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZRANGE"}, {Str: "lb"}, {Str: "0"}, {Str: "-1"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		result := buf.String()
		require.Contains(t, result, "Alice")
		require.Contains(t, result, "Charlie")
		require.Contains(t, result, "Bob")
		require.Contains(t, result, "*3\r\n")
	})

	t.Run("ZRange subset", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)
		store := storage.NewStore()

		for _, args := range [][]protocol.RESPValue{
			{{Str: "ZADD"}, {Str: "lb"}, {Str: "100"}, {Str: "Alice"}},
			{{Str: "ZADD"}, {Str: "lb"}, {Str: "200"}, {Str: "Bob"}},
			{{Str: "ZADD"}, {Str: "lb"}, {Str: "150"}, {Str: "Charlie"}},
		} {
			ctx := CommandContext{Args: args, Writer: protocol.NewWriter(w), Store: store}
			router.Handle(&ctx)
			w.Flush()
			buf.Reset()
			w.Reset(&buf)
		}

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZRANGE"}, {Str: "lb"}, {Str: "0"}, {Str: "1"},
			},
			Writer: protocol.NewWriter(w),
			Store:  store,
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Contains(t, buf.String(), "*2\r\n")
	})

	t.Run("ZRange missing key", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		ctx := CommandContext{
			Args: []protocol.RESPValue{
				{Str: "ZRANGE"}, {Str: "missing"}, {Str: "0"}, {Str: "-1"},
			},
			Writer: protocol.NewWriter(w),
			Store:  storage.NewStore(),
		}
		router.Handle(&ctx)
		ctx.Writer.Flush()

		require.Equal(t, "*0\r\n", buf.String())
	})
}

type mockRewriter struct {
	called atomic.Bool
}

func (mr *mockRewriter) Rewrite(storage *storage.Store) error {
	mr.called.Store(true)
	return nil
}

func TestBgRewriteAOF(t *testing.T) {
	router := NewRouter()
	router.Register("BGREWRITEAOF", cmdBgRewriteAOF)

	t.Run("correct args", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		mock := &mockRewriter{}

		ctx := CommandContext{
			Args:     []protocol.RESPValue{{Str: "BGREWRITEAOF", Type: protocol.BulkString}},
			Writer:   protocol.NewWriter(w),
			Store:    storage.NewStore(),
			Rewriter: mock,
		}
		err := router.Handle(&ctx)
		ctx.Writer.Flush()

		require.NoError(t, err)
		require.Equal(t, "+Background append only file rewriting started\r\n", buf.String())
		time.Sleep(time.Millisecond)
		require.True(t, mock.called.Load())
	})

	t.Run("wrong args number", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		mock := &mockRewriter{}

		ctx := CommandContext{
			Args:     []protocol.RESPValue{{Str: "BGREWRITEAOF", Type: protocol.BulkString}, {Str: "second_arg", Type: protocol.BulkString}},
			Writer:   protocol.NewWriter(w),
			Store:    storage.NewStore(),
			Rewriter: mock,
		}

		err := router.Handle(&ctx)
		ctx.Writer.Flush()

		require.NoError(t, err)
		require.Contains(t, buf.String(), "ERR")
		time.Sleep(time.Millisecond)
		require.False(t, mock.called.Load())
	})

	t.Run("nil rewriter", func(t *testing.T) {
		buf := bytes.Buffer{}
		w := bufio.NewWriter(&buf)

		mock := &mockRewriter{}

		ctx := CommandContext{
			Args:     []protocol.RESPValue{{Str: "BGREWRITEAOF", Type: protocol.BulkString}},
			Writer:   protocol.NewWriter(w),
			Store:    storage.NewStore(),
			Rewriter: nil,
		}

		err := router.Handle(&ctx)
		ctx.Writer.Flush()

		require.NoError(t, err)
		require.Equal(t, "-nil rewriter\r\n", buf.String())
		time.Sleep(time.Millisecond)
		require.False(t, mock.called.Load())
	})
}
