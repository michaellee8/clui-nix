package clui

import "io"

// Provider represents an interface that each clui backend should provide
type Provider interface {

	// SetDir sets the current working directory of the required command
	SetDir(string)

	// SetInput sets the input stream for Stdin
	SetInput(io.Reader)

	// SetOutput sets the output stream for Stdout and Stderr
	SetOutput(io.Writer)

	// SetCompOptHandler sets the handler for completion options
	SetCompOptHandler(CompletionInfoHandler)

	// Start starts the backend for user input, will return error only
	// if the starting process failed. Should not return until the underlying
	// process exited.
	Start() error
}
