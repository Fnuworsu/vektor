package resp

import (
	"fmt"
	"io"
)

func WriteError(w io.Writer, err error) error {
	_, writeErr := fmt.Fprintf(w, "-ERR %s\r\n", err.Error())
	return writeErr
}

func WriteSimpleString(w io.Writer, s string) error {
	_, writeErr := fmt.Fprintf(w, "+%s\r\n", s)
	return writeErr
}

func WriteBulkString(w io.Writer, b []byte) error {
	if b == nil {
		_, writeErr := fmt.Fprint(w, "$-1\r\n")
		return writeErr
	}
	_, writeErr := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(b), b)
	return writeErr
}

func WriteInteger(w io.Writer, n int) error {
	_, writeErr := fmt.Fprintf(w, ":%d\r\n", n)
	return writeErr
}
