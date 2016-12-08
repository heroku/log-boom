package syslog

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

// Syslog scanning errors
var (
	ErrNotRFC6587 = errors.New("Not RFC6587 Formatted Syslog")
)

// ScanRFC6587 does stuff
func ScanRFC6587(data []byte, atEOF bool) (int, []byte, error) {
	mark := 0
	for ; mark < len(data); mark++ {
		if data[mark] == ' ' {
			break
		}
	}

	for i := mark; i < len(data); i++ {
		if data[i] == '<' {
			offset, err := strconv.Atoi(string(data[0:mark]))
			if err != nil {
				return 0, nil, err
			}
			token := data[:mark+offset+1]
			return len(token), token, nil
		}
	}

	if atEOF && len(data) > mark {
		return 0, nil, ErrNotRFC6587
	}

	// Request more data.
	return mark, nil, nil
}

// Scan scans the reader for count RFC6587 formatted syslog entries.
func Scan(r io.Reader, count int64) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Split(ScanRFC6587)

	lines := make([]string, 0, count)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
