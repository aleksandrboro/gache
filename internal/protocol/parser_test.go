package protocol

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseSimpleString(t *testing.T) {
	testcases := []struct {
		Name     string
		Value    string
		Expected string
		IsValid  bool
	}{
		{
			Name:     "valid string",
			Value:    "+OK\r\n",
			Expected: "OK",
			IsValid:  true,
		},
		{
			Name:    "invalid string",
			Value:   "+OK\r",
			IsValid: false,
		},
		{
			Name:     "empty simple string",
			Value:    "+\r\n",
			Expected: "",
			IsValid:  true,
		},
	}

	t.Parallel()
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := parseHelper(t, tc.Value)
			if tc.IsValid {
				require.NoError(t, err)
				require.Equal(t, tc.Expected, resp.Str)
				require.Equal(t, SimpleString, resp.Type)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func Test_parseError(t *testing.T) {
	testcases := []struct {
		Name     string
		Value    string
		Expected string
		IsValid  bool
	}{
		{
			Name:     "valid error",
			Value:    "-error\r\n",
			Expected: "error",
			IsValid:  true,
		},
		{
			Name:    "invalid error",
			Value:   "-error\r",
			IsValid: false,
		},
		{
			Name:    "empty error",
			Value:   "",
			IsValid: false,
		},
	}

	t.Parallel()
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := parseHelper(t, tc.Value)
			if tc.IsValid {
				require.NoError(t, err)
				require.Equal(t, tc.Expected, resp.Str)
				require.Equal(t, Error, resp.Type)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func Test_parseInteger(t *testing.T) {
	testcases := []struct {
		Name     string
		Value    string
		Expected int64
		IsValid  bool
	}{
		{
			Name:     "valid integer",
			Value:    ":1\r\n",
			Expected: 1,
			IsValid:  true,
		},
		{
			Name:    "invalid integer",
			Value:   ":1\r",
			IsValid: false,
		},
		{
			Name:    "empty integer",
			Value:   "",
			IsValid: false,
		},
		{
			Name:     "zero",
			Value:    ":0\r\n",
			Expected: 0,
			IsValid:  true,
		},
		{
			Name:     "negative",
			Value:    ":-1\r\n",
			Expected: -1,
			IsValid:  true,
		},
		{
			Name:     "large number",
			Value:    ":1000000\r\n",
			Expected: 1000000,
			IsValid:  true,
		},
		{
			Name:    "invalid number",
			Value:   ":abc\r\n",
			IsValid: false,
		},
	}

	t.Parallel()
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := parseHelper(t, tc.Value)
			if tc.IsValid {
				require.NoError(t, err)
				require.Equal(t, tc.Expected, resp.Int)
				require.Equal(t, Integer, resp.Type)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func Test_parseBulkString(t *testing.T) {
	testcases := []struct {
		Name     string
		Value    string
		Expected string
		IsValid  bool
	}{
		{
			Name:     "valid string",
			Value:    "$11\r\nhello world\r\n",
			Expected: "hello world",
			IsValid:  true,
		},
		{
			Name:    "invalid string",
			Value:   "$11\r\nhello world\r",
			IsValid: false,
		},
		{
			Name:    "empty string",
			Value:   "",
			IsValid: false,
		},
		{
			Name:     "empty bulk string",
			Value:    "$0\r\n\r\n",
			Expected: "",
			IsValid:  true,
		},
		{
			Name:     "null bulk string",
			Value:    "$-1\r\n",
			Expected: "",
			IsValid:  true,
		},
		{
			Name:     "utf-8 string",
			Value:    "$12\r\nпривет\r\n",
			Expected: "привет",
			IsValid:  true,
		},
		{
			Name:    "invalid length",
			Value:   "$-2\r\n",
			IsValid: false,
		},
	}

	t.Parallel()
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := parseHelper(t, tc.Value)
			if tc.IsValid {
				if tc.Name == "null bulk string" {
					require.True(t, resp.IsNull)
				} else {
					require.False(t, resp.IsNull)
				}
				require.NoError(t, err)
				require.Equal(t, tc.Expected, resp.Str)
				require.Equal(t, BulkString, resp.Type)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func Test_parseArray(t *testing.T) {
	testcases := []struct {
		Name     string
		Value    string
		Count    int
		Expected []string
		IsValid  bool
	}{
		{
			Name:     "valid array",
			Value:    "*5\r\n$3\r\nSET\r\n$3\r\nkey\r\n$11\r\nhello world\r\n$2\r\nEX\r\n$2\r\n10\r\n",
			Expected: []string{"SET", "key", "hello world", "EX", "10"},
			IsValid:  true,
		},
		{
			Name:    "null array",
			Value:   "*-1\r\n",
			Count:   -1,
			IsValid: true,
		},
		{
			Name:    "invalid array",
			Value:   "*5\r\n$3\r\nSET\r\n$3\r\nkey\r\n$11\r\nhello world\r\n$2\r\nEX\r\n$2\r\n10\r",
			IsValid: false,
		},
		{
			Name:    "empty input",
			Value:   "",
			IsValid: false,
		},
		{
			Name:     "empty array",
			Value:    "*0\r\n",
			Count:    0,
			Expected: []string{},
			IsValid:  true,
		},
	}

	t.Parallel()
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := parseHelper(t, tc.Value)
			if tc.IsValid {
				if tc.Count == -1 {
					require.True(t, resp.IsNull)
				} else {
					require.False(t, resp.IsNull)
					require.Len(t, resp.Array, len(tc.Expected))

					for i, expected := range tc.Expected {
						require.Equal(t, expected, resp.Array[i].Str)
					}
				}
				require.NoError(t, err)
				require.Equal(t, Array, resp.Type)
			} else {
				require.Error(t, err)
				require.Equal(t, 0, len(resp.Array))
			}
		})
	}
}

func parseHelper(t *testing.T, input string) (RESPValue, error) {
	t.Helper()
	reader := bufio.NewReader(strings.NewReader(input))
	parser := NewParser(reader)
	return parser.Parse()
}
