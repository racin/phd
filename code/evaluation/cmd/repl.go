package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/shlex"
	"golang.org/x/term"
)

type repl struct {
	term *term.Terminal
}

func newRepl() *repl {
	r := &repl{
		term: term.NewTerminal(struct {
			io.Reader
			io.Writer
		}{os.Stdin, os.Stderr}, "> "),
	}
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err == nil {
		_ = r.term.SetSize(width, height)
	}
	return r
}

// ReadLine reads a line from the terminal in raw mode.
//
// FIXME: ReadLine currently does not work with arrow keys on windows for some reason
// See: https://stackoverflow.com/questions/58237670/terminal-raw-mode-does-not-support-arrows-on-windows
func (r repl) ReadLine() (line string, err error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("failed to set terminal mode: %w", err)
	}

	defer func() {
		rerr := term.Restore(fd, oldState)
		if rerr != nil && err == nil {
			err = fmt.Errorf("failed to restore terminal mode: %w", rerr)
		}
	}()

	return r.term.ReadLine()
}

// Repl runs an interactive Read-eval-print loop.
func Repl(ctx context.Context) {
	r := newRepl()

	fmt.Println("Interactive mode enabled.")

	for {
		l, err := r.ReadLine()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read line: %v\n", err)
			exitCode = 1
			return
		}
		args, err := shlex.Split(l)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to split command: %v\n", err)
			exitCode = 1
			return
		}
		if len(args) < 1 {
			continue
		}

		switch args[0] {
		case "exit":
			fallthrough
		case "quit":
			return
		case "help":
			rootCmd.Usage()
		default:
			rootCmd.SetArgs(args)
			err := rootCmd.ExecuteContext(ctx)
			var errBatch errBatch
			if errors.As(err, &errBatch) {
				Batch(ctx, errBatch.batch, experimentRetries)
			} else if err != nil {
				printError(err)
				exitCode = 1
			} else {
				exitCode = 0
			}
		}
	}
}

func Batch(ctx context.Context, batchFile string, retries int) {
	f, err := os.Open(batchFile)
	if err != nil {
		printError(err)
		exitCode = 1
		return
	}
	defer func() {
		err = f.Close()
		if err != nil {
			printError(err)
			exitCode = 1
		}
	}()

	rd := bufio.NewReader(f)

	for {
		if ctx.Err() != nil {
			return
		}

		line, err := rd.ReadString('\n')
		if errors.Is(err, io.EOF) {
			return
		} else if err != nil {
			printError(err)
			exitCode = 1
			return
		}

		args, err := shlex.Split(strings.TrimSpace(line))

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to split command: %v\n", err)
			exitCode = 1
			return
		}

		if len(args) == 0 {
			continue
		}

		rootCmd.SetArgs(args)

		for i := 1; i <= retries; i++ {
			fmt.Printf("Executing Command (%d/%d) %v\n", i, retries, args)
			err = rootCmd.ExecuteContext(ctx)
			if err != nil {
				printError(err)
				exitCode = 1
			} else {
				exitCode = 0
				break
			}
		}
	}
}
