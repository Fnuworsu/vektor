package resp

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
)

func ParseCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")
	line = strings.TrimSuffix(line, "\n")

	if len(line) == 0 {
		return nil, errors.New("empty command")
	}

	if line[0] != '*' {
		return strings.Fields(line), nil
	}

	argc, err := strconv.Atoi(line[1:])
	if err != nil || argc <= 0 {
		return nil, errors.New("invalid multibulk length")
	}

	args := make([]string, argc)
	for i := 0; i < argc; i++ {
		lenLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		lenLine = strings.TrimSuffix(lenLine, "\r\n")
		lenLine = strings.TrimSuffix(lenLine, "\n")

		if len(lenLine) == 0 || lenLine[0] != '$' {
			return nil, errors.New("expected bulk string size")
		}

		size, err := strconv.Atoi(lenLine[1:])
		if err != nil || size < 0 {
			return nil, errors.New("invalid bulk string size")
		}

		buf := make([]byte, size+2)
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			return nil, err
		}

		args[i] = string(buf[:size])
	}

	return args, nil
}
