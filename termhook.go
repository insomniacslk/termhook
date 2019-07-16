package termhook

import (
	"fmt"
	"io"
	"log"
	"os"
	"syscall"

	"github.com/pkg/term"
	"github.com/pkg/term/termios"
)

// LineHandler defines the function that is called at each line of output from
// the terminal. The boolean indicates whether the terminal reader should return.
type LineHandler func(w io.Writer, line []byte) (bool, error)

// Hook is the terminal hook type.
type Hook struct {
	term        *term.Term
	Port        string
	Speed       int
	ReadOnly    bool
	sigs        chan os.Signal
	lineHandler LineHandler
}

// NewHook initializes and returns a new Hook object.
func NewHook(port string, speed int, handler LineHandler) (*Hook, error) {
	h := Hook{
		Port:        port,
		Speed:       speed,
		sigs:        make(chan os.Signal, 1),
		lineHandler: handler,
	}
	if h.lineHandler == nil {
		h.lineHandler = h.defaultLineHandler
	}
	return &h, nil
}

// Run starts the terminal hook handler. This function is blocking.
func (h *Hook) Run() error {
	t, err := term.Open(h.Port, term.Speed(h.Speed), term.RawMode)
	if err != nil {
		return err
	}
	h.term = t
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.handleSignals(t)
	}()
	if !h.ReadOnly {
		go func() {
			errCh <- h.handleInput(t)
		}()
	}

	buf := make([]byte, 1024)
	for {
		n, err := t.Read(buf)
		if err != nil {
			return err
		}
		stop, err := h.lineHandler(t, buf[:n])
		if err != nil {
			log.Printf("Error handling line: %v", err)
			return err
		}
		if stop {
			break
		}
	}
	return nil
}

// Close closes the terminal hook.
func (h *Hook) Close() error {
	return h.term.Close()
}

// handleSignals handles the signals received by the process.
func (h *Hook) handleSignals(w io.Writer) error {
	ctrlC := []byte{3}
	sig := <-h.sigs
	if sig == syscall.SIGINT {
		if _, err := w.Write(ctrlC); err != nil {
			return err
		}
	}
	return nil
}

// handleInput handles the input coming from stdin
func (h *Hook) handleInput(w io.Writer) error {
	// set stdin unbuffered
	var a syscall.Termios
	if err := termios.Tcgetattr(uintptr(syscall.Stdin), (&a)); err != nil {
		return err
	}
	termios.Cfmakeraw((*syscall.Termios)(&a))
	if err := termios.Tcsetattr(uintptr(syscall.Stdin), termios.TCSANOW, &a); err != nil {
		return err
	}

	b := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(b)
		if err != nil {
			return err
		}
		if _, err := w.Write(b[:n]); err != nil {
			return err
		}
	}
}

// defaultLineHandler implements LineHandler.
func (h *Hook) defaultLineHandler(w io.Writer, line []byte) (bool, error) {
	fmt.Print(string(line))
	return false, nil
}
