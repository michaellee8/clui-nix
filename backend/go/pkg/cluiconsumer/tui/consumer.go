package tui

import (
	"fmt"
	"github.com/kr/pty"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
	protoclui "github.com/michaellee8/clui-nix/backend/go/pkg/proto/clui"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var esc = "\033"

func getEscCode(n int, op string) string {
	if n < 0 {
		return fmt.Sprintf("%s[%s", esc, op)
	}
	return fmt.Sprintf("%s[%d%s", esc, n, op)
}

func printEscCode(n int, op string) {
	fmt.Print(getEscCode(n, op))
}

// Consumer implements clui.Consumer for the tui frontend, it is adhoc and
// is intended for demostration and testing purpose only, should not been
// used in production, please use websocket consumer instead.
type Consumer struct {
	dir          string
	input        io.Reader
	output       io.Writer
	handler      clui.CompletionInfoHandler
	winsizeChan  chan pty.Winsize
	osSignalChan chan os.Signal
}

func (c *Consumer) Init() (err error) {
	if c.dir, err = os.Getwd(); err != nil {
		return errors.Wrap(err, "cannot get pwd")
	}

	c.input = os.Stdin
	c.output = os.Stdout

	c.osSignalChan = make(chan os.Signal, 1)
	signal.Notify(c.osSignalChan, syscall.SIGWINCH)

	c.winsizeChan = make(chan pty.Winsize)

	go func() {
		for range c.osSignalChan {
			if winsize, err := pty.GetsizeFull(os.Stdin); err != nil {
				logrus.Error(errors.Wrap(err, "cannot get terminal size"))
			} else {
				c.winsizeChan <- *winsize
			}
		}
	}()

	c.osSignalChan <- syscall.SIGWINCH // initial resize

	return

}

func (c *Consumer) Handle(ci *protoclui.CompletionInfo) {

	logrus.Tracef("tui handling completion on bufl %d, count %d", ci.BufferLength, len(ci.Entries))
	if len(ci.Entries) == 0 {
		// ignore if no entries
		return
	}

	// save cursor pos
	printEscCode(-1, "s")

	// move to our place to write first completion result
	printEscCode(-1, "H")
	printEscCode(-1, "K")
	printEscCode(1, "m")
	printEscCode(37, "m")
	printEscCode(44, "m")

	fmt.Printf("%s %s %d %d", ci.Entries[0].Suggestion, ci.Entries[0].Description, ci.Line, ci.Col)

	// restore cursor pos
	printEscCode(-1, "u")

	printEscCode(0, "m")
}

func (c *Consumer) Dir() string {
	return c.dir
}

func (c *Consumer) Input() io.Reader {
	return c.input
}

func (c *Consumer) Output() io.Writer {
	return c.output
}

func (c *Consumer) CompOptHandler() clui.CompletionInfoHandler {
	return c
}

func (c *Consumer) WinsizeChan() chan pty.Winsize {
	return c.winsizeChan
}

func (c *Consumer) OnStart() {
}
