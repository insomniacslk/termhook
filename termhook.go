/* SPDX-License-Identifier: BSD-2-Clause */

package termhook

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/term"
	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

// LineHandler defines the function that is called at each line of output from
// the terminal. The boolean indicates whether the terminal reader should return.
type LineHandler func(w io.Writer, line []byte) (bool, error)

// Hook is the terminal hook type.
type Hook struct {
	term        *term.Term
	port        string
	speed       int
	handleStdin bool
	lineHandler LineHandler
}

// NewHook initializes and returns a new Hook object.
func NewHook(port string, speed int, handleStdin bool, handler LineHandler) (*Hook, error) {
	h := Hook{
		port:        port,
		speed:       speed,
		handleStdin: handleStdin,
		lineHandler: handler,
	}
	if h.lineHandler == nil {
		h.lineHandler = h.defaultLineHandler
	}
	return &h, nil
}

// Run starts the terminal hook handler. This function is blocking.
func (h *Hook) Run() error {
	t, err := term.Open(h.port, term.Speed(h.speed), term.RawMode)
	if err != nil {
		return err
	}
	h.term = t

	var wg sync.WaitGroup

	var routineError error
	var routineErrorOnce sync.Once
	processRoutineError := func(err error) {
		if err == nil {
			return
		}
		routineErrorOnce.Do(func() {
			routineError = err
		})
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT)

	var stopSignalsOnce sync.Once
	stopReceivingSignals := func() {
		stopSignalsOnce.Do(func() {
			signal.Stop(signalCh)
			close(signalCh)
		})
	}
	defer stopReceivingSignals()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := h.handleSignals(signalCh, t)
		processRoutineError(err)
	}()

	if h.handleStdin {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := h.handleInput(t)
			processRoutineError(err)
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

	stopReceivingSignals()
	wg.Wait()
	return routineError
}

// Close closes the terminal hook.
func (h *Hook) Close() error {
	if h.term == nil {
		return nil
	}
	return h.term.Close()
}

// handleSignals handles the signals received by the process.
func (h *Hook) handleSignals(signalCh <-chan os.Signal, w io.Writer) error {
	ctrlC := []byte{3}
	for sig := range signalCh {
		if sig == syscall.SIGINT {
			if _, err := w.Write(ctrlC); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// handleInput handles the input coming from stdin
func (h *Hook) handleInput(w io.Writer) error {
	// set stdin unbuffered
	var a unix.Termios
	if err := termios.Tcgetattr(uintptr(syscall.Stdin), &a); err != nil {
		return err
	}
	termios.Cfmakeraw(&a)
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
