package tui

import (
	"io"

	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
)

type Consumer struct {
}

// Dir should return the current working directory that the Consumer intends
// to have, it is guaranteed the created process will have such cwd
func (c *Consumer) Dir() string {
	panic("not implemented") // TODO: Implement
}

// Input should return an io.Reader that the Consumer wants the provider to
// read input from
func (c *Consumer) Input() io.Reader {
	panic("not implemented") // TODO: Implement
}

// Output should return an io.Writer that the Consumer wants the provider to
// write output to
func (c *Consumer) Output() io.Writer {
	panic("not implemented") // TODO: Implement
}

// CompOptHandler should return an CompletionInfoHandler that will receives
// all completion information provided by the Provider, it must be safe for
// multiple concurrent invocation of Handle()
func (c *Consumer) CompOptHandler() clui.CompletionInfoHandler {
	panic("not implemented") // TODO: Implement
}

// OnStart is a callback that will be called after the Provider has started
// the backing process and have all preparation done successfully. It should
// only be called once
func (c *Consumer) OnStart() {
	panic("not implemented") // TODO: Implement
}
