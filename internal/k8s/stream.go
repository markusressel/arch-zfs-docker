package k8s

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
)

const (
	// maxLogLine caps a single log line; pacman redraws progress bars with \r, so one "line"
	// can grow large. Longer lines are cut instead of ending the stream.
	maxLogLine      = 1 << 20
	truncatedMarker = " [line truncated]"
)

// streamLines delivers body line by line (without line endings) until it ends or ctx is
// cancelled. Cancelling closes body, so an upstream connection is released right away.
func streamLines(ctx context.Context, body io.ReadCloser) <-chan string {
	lines := make(chan string, 100)
	go func() {
		defer close(lines)
		defer body.Close()
		stop := context.AfterFunc(ctx, func() { body.Close() })
		defer stop()

		r := bufio.NewReader(body)
		for {
			line, err := readLine(r)
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case lines <- line:
			}
		}
	}()
	return lines
}

// readLine returns the next line without its line ending, cut to maxLogLine.
func readLine(r *bufio.Reader) (string, error) {
	var buf []byte
	truncated := false
	for {
		chunk, isPrefix, err := r.ReadLine()
		if err != nil {
			return "", err // ReadLine returns an error only when there is no data
		}
		room := maxLogLine - len(buf)
		if len(chunk) > room {
			chunk, truncated = chunk[:room], true
		}
		buf = append(buf, chunk...)
		if !isPrefix {
			break
		}
	}
	line := strings.TrimRight(string(buf), "\r")
	if truncated {
		line += truncatedMarker
	}
	return line, nil
}

// streamFromFile replays a persisted build log.
func streamFromFile(ctx context.Context, path string) (<-chan string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return streamLines(ctx, f), nil
}
