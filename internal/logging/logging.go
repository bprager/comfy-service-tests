package logging

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type prefixedWriter struct {
	service string
	out     io.Writer
}

func (w *prefixedWriter) Write(p []byte) (int, error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	prefix := fmt.Sprintf("%s %s ", timestamp, w.service)
	lines := bytes.Split(p, []byte{'\n'})

	var buf bytes.Buffer
	for i, line := range lines {
		if len(line) == 0 && i == len(lines)-1 {
			break
		}
		buf.WriteString(prefix)
		buf.Write(line)
		buf.WriteByte('\n')
	}

	if buf.Len() == 0 {
		return len(p), nil
	}
	if _, err := w.out.Write(buf.Bytes()); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Setup configures the default logger to write to stdout and a log file.
func Setup(serviceName, logDir string) (*os.File, error) {
	if logDir == "" {
		logDir = ".log"
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	logPath := filepath.Join(logDir, serviceName+".log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	log.SetOutput(&prefixedWriter{
		service: serviceName,
		out:     io.MultiWriter(os.Stdout, file),
	})
	log.SetFlags(0)
	log.SetPrefix("")
	return file, nil
}
