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
// the terminal.
type LineHandler func(w io.Writer, line []byte) error

// Hook is the terminal hook type.
type Hook struct {
	term        *term.Term
	Port        string
	Speed       int
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
	go h.HandleInputAndSignals(errCh, t)

	buf := make([]byte, 1024)
	for {
		n, err := t.Read(buf)
		if err != nil {
			return err
		}
		if err := h.lineHandler(t, buf[:n]); err != nil {
			log.Printf("Error handling line: %v", err)
			return err
		}
	}
}

// Close closes the terminal hook.
func (h *Hook) Close() error {
	return h.term.Close()
}

// HandleInputAndSignals handles the input coming from stdin, and the signals
// sent by the user.
func (h *Hook) HandleInputAndSignals(errCh chan<- error, w io.Writer) {
	// set stdin unbuffered
	var a syscall.Termios
	if err := termios.Tcgetattr(uintptr(syscall.Stdin), (&a)); err != nil {
		errCh <- err
		return
	}
	termios.Cfmakeraw((*syscall.Termios)(&a))
	if err := termios.Tcsetattr(uintptr(syscall.Stdin), termios.TCSANOW, &a); err != nil {
		errCh <- err
		return
	}

	b := make([]byte, 1)
	ctrlC := []byte{3}
	for {
		select {
		case sig := <-h.sigs:
			if sig == syscall.SIGINT {
				if _, err := w.Write(ctrlC); err != nil {
					errCh <- err
					return
				}
			}
		default:
			n, err := os.Stdin.Read(b)
			if err != nil {
				errCh <- err
				return
			}
			if _, err := w.Write(b[:n]); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func (h *Hook) defaultLineHandler(w io.Writer, line []byte) error {
	fmt.Print(string(line))
	return nil
}
