package tui

import (
	"fmt"
	"io"
	"os"

	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
	protoclui "github.com/michaellee8/clui-nix/backend/go/pkg/proto/clui"
	"github.com/pkg/errors"
)

var esc = "\033"

func getEscCode(n int, op string) string {
	if n == 0 {
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
	dir     string
	input   io.Reader
	output  io.Writer
	handler clui.CompletionInfoHandler
}

func (c *Consumer) Init() (err error) {
	if c.dir, err = os.Getwd(); err != nil {
		return errors.Wrap(err, "cannot get pwd")
	}

	c.input = os.Stdin
	c.output = os.Stdout

	return

}

func (c *Consumer) Handle(ci protoclui.CompletionInfo) {

	if len(ci.Entries) == 0 {
		// ignore if no entries
		return
	}

	// save cursor pos
	printEscCode(0, "s")

	// move to our place to write first completion result
	printEscCode(1, "E")

	fmt.Printf("%s %s %d %d\n", ci.Entries[0].Suggestion, ci.Entries[0].Description, ci.Line, ci.Col)

	// restore cursor pos
	printEscCode(0, "u")
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
	panic("not implemented") // TODO: Implement
}

func (c *Consumer) OnStart() {
}
