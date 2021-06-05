package clui

import "io"

// Consumer represents any frontend that would like to consume the clui interface
// for example an Websocket server or a terminal-based UI interface
type Consumer interface {

	// Dir should return the current working directory that the Consumer intends
	// to have, it is guaranteed the created process will have such cwd
	Dir() string

	// Input should return an io.Reader that the Consumer wants the provider to
	// read input from
	Input() io.Reader

	// Output should return an io.Writer that the Consumer wants the provider to
	// write output to
	Output() io.Writer

	// CompOptHandler should return an CompletionInfoHandler that will receives
	// all completion information provided by the Provider, it must be safe for
	// multiple concurrent invocation of Handle()
	CompOptHandler() CompletionInfoHandler

	// OnStart is a callback that will be called right before the Provider starts
	// the backing process and have all preparation done successfully. It should
	// only be called once
	OnStart()
}
