package request

import (
	"bufio"
	"bytes"
	"io"
)

// Scans for `\r\n`. Adopted from [bufio.ScanLines].
var ScanCRLF = func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.Index(data, registeredNurse); i >= 0 {
		// full line
		return i + 2, data[:i], nil
	}

	// if atEOF, return remaining data
	if atEOF {
		return len(data), data, nil
	}

	// request more data
	return 0, nil, nil
}

func getCRLFScanner(reader io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(reader)
	scanner.Split(ScanCRLF)

	return scanner
}
