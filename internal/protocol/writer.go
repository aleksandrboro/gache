package protocol

import (
	"bufio"
	"fmt"
)

type Writer struct {
	w *bufio.Writer
}

func NewWriter(w *bufio.Writer) *Writer {
	return &Writer{
		w: w,
	}
}

func (w *Writer) WriteSimpleString(s string) error {
	_, err := fmt.Fprintf(w.w, "+%s\r\n", s)

	return err
} // +s\r\n

func (w *Writer) WriteError(msg string) error {
	_, err := fmt.Fprintf(w.w, "-%s\r\n", msg)

	return err
} // -msg\r\n

func (w *Writer) WriteInteger(n int64) error {
	_, err := fmt.Fprintf(w.w, ":%d\r\n", n)

	return err
} // :n\r\n

func (w *Writer) WriteBulkString(s string) error {
	_, err := fmt.Fprintf(w.w, "$%d\r\n%s\r\n", len(s), s)

	return err
} // $len\r\ns\r\n

func (w *Writer) WriteNull() error {
	_, err := fmt.Fprint(w.w, "$-1\r\n")

	return err
} // $-1\r\n

func (w *Writer) WriteArray(values []RESPValue) error {
	if _, err := fmt.Fprintf(w.w, "*%d\r\n", len(values)); err != nil {
		return err
	}

	for _, v := range values {
		var err error
		switch v.Type {
		case SimpleString:
			err = w.WriteSimpleString(v.Str)
		case Error:
			err = w.WriteError(v.Str)
		case Integer:
			err = w.WriteInteger(v.Int)
		case BulkString:
			if v.IsNull {
				err = w.WriteNull()
			} else {
				err = w.WriteBulkString(v.Str)
			}
		case Array:
			err = w.WriteArray(v.Array)
		default:
			err = fmt.Errorf("unknown RESP type: %v", v.Type)
		}

		if err != nil {
			return err
		}
	}

	return nil
} // *N\r\n...

func (w *Writer) Flush() error {
	return w.w.Flush()
}
