package resp

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseCommandInline(t *testing.T) {
	input := "GET foo\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd, err := ParseCommand(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmd) != 2 || cmd[0] != "GET" || cmd[1] != "foo" {
		t.Fatalf("unexpected command: %v", cmd)
	}
}

func TestParseCommandMultibulk(t *testing.T) {
	input := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	cmd, err := ParseCommand(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmd) != 3 || cmd[0] != "SET" || cmd[1] != "foo" || cmd[2] != "bar" {
		t.Fatalf("unexpected command: %v", cmd)
	}
}
