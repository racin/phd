package cli

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// ProgressWriter writes a simple progress bar.
type ProgressWriter struct {
	mut       sync.Mutex
	startTime time.Time
	wr        io.Writer
	message   string
	total     int
	progress  int
}

func NewProgress(wr io.Writer, message string, total int) (*ProgressWriter, error) {
	pw := &ProgressWriter{
		wr:        wr,
		message:   message,
		total:     total,
		startTime: time.Now(),
	}

	err := pw.write()
	if err != nil {
		return nil, err
	}

	return pw, nil
}

func (pw *ProgressWriter) SetMessage(message string) error {
	pw.mut.Lock()
	defer pw.mut.Unlock()
	pw.message = message
	return pw.write()
}

func (pw *ProgressWriter) SetTotal(total int) error {
	pw.mut.Lock()
	defer pw.mut.Unlock()
	pw.total = total
	return pw.write()
}

func (pw *ProgressWriter) SetProgress(progress int) error {
	pw.mut.Lock()
	defer pw.mut.Unlock()
	pw.progress = progress
	return pw.write()
}

func (pw *ProgressWriter) Increment() error {
	pw.mut.Lock()
	defer pw.mut.Unlock()
	pw.progress++
	return pw.write()
}

func (pw *ProgressWriter) Done() error {
	_, err := io.WriteString(pw.wr, fmt.Sprintf(" - took %s\n", time.Since(pw.startTime)))
	return err
}

func (pw *ProgressWriter) write() error {
	_, err := io.WriteString(pw.wr, fmt.Sprintf("\r%s(%d/%d)", pw.message, pw.progress, pw.total))
	return err
}
