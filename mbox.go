// Based on: https://github.com/emersion/go-mbox
//
// "THE BEER-WARE LICENSE" (Revision 42):
// <tobias.rehbein@web.de> wrote this file. As long as you retain this notice
// you can do whatever you want with this stuff. If we meet some day, and you
// think this stuff is worth it, you can buy me a beer in return.
//
//	Tobias Rehbein
package mbox

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"regexp"
)

// Reader reads an mbox archive.
type Reader struct {
	r  *bufio.Reader
	mr *messageReader
}

type messageReader struct {
	r              *bufio.Reader
	next           bytes.Buffer
	atEOF          bool
	atSeparator    bool
	atMiddleOfLine bool
}

var (
	ErrInvalidFormat = errors.New("invalid mbox format")
	reHeader         = regexp.MustCompile(`(?m)^[a-zA-Z0-9]{1,}(([-][a-zA-Z0-9]{1,})?)*\s*:`)
)

// NewReader returns a new Reader to read messages from mbox file format data
// provided by io.Reader r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: bufio.NewReader(r)}
}

// NextMessage returns the next message text (containing both the header and the
// body). It will return io.EOF if there are no messages left.
func (r *Reader) NextMessage() (io.Reader, error) {
	if r.mr == nil {
		for {
			b, isPrefix, err := r.r.ReadLine()
			if err != nil {
				return nil, err
			}

			// Discard the rest of the line.
			for isPrefix {
				_, isPrefix, err = r.r.ReadLine()
				if err != nil {
					return nil, err
				}
			}

			if len(b) == 0 {
				continue
			}

			if isFromLine(r.r, b) {
				break
			} else {
				return nil, ErrInvalidFormat
			}
		}
	} else {
		if _, err := io.Copy(io.Discard, r.mr); err != nil {
			return nil, err
		}

		if r.mr.atEOF {
			return nil, io.EOF
		}
	}

	r.mr = &messageReader{r: r.r}

	return r.mr, nil
}

func (mr *messageReader) Read(p []byte) (int, error) {
	if mr.atEOF || mr.atSeparator {
		return 0, io.EOF
	}

	if mr.next.Len() == 0 {
		b, isPrefix, err := mr.r.ReadLine()
		if err != nil {
			mr.atEOF = true
			return 0, err
		}

		if !mr.atMiddleOfLine {
			if isFromLine(mr.r, b) {
				mr.atSeparator = true
				return 0, io.EOF
			} else if len(b) == 0 {
				// Check if the next line is separator. In such case the new
				// line should not be written to not have double new line.
				b, isPrefix, err = mr.r.ReadLine()
				if err != nil {
					mr.atEOF = true
					return 0, err
				}

				if isFromLine(mr.r, b) {
					mr.atSeparator = true
					return 0, io.EOF
				}

				mr.next.Write([]byte("\r\n"))
			}
		}

		mr.next.Write(b)
		if !isPrefix {
			mr.next.Write([]byte("\r\n"))
		}

		mr.atMiddleOfLine = isPrefix
	}

	return mr.next.Read(p)
}

func isFromLine(r *bufio.Reader, currentLine []byte) bool {
	if !bytes.HasPrefix(currentLine, []byte("From ")) {
		return false
	}

	b, _ := r.Peek(2048)
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return false
	}

	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))

	mimec := 0
	for _, cl := range bytes.Split(b, []byte("\n")) {
		cl = bytes.TrimSpace(cl)

		if len(cl) > 0 {
			if reHeader.Match(cl) {
				mimec++
			}
		} else {
			return mimec >= 2
		}
	}

	return mimec >= 2
}
