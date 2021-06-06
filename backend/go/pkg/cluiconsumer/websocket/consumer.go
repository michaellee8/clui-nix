package wsconsumer

import (
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
	protoclui "github.com/michaellee8/clui-nix/backend/go/pkg/proto/clui"
	"github.com/pkg/errors"
)

// reference: https://github.com/gorilla/websocket/blob/master/examples/echo/server.go

// Consumer implements clui.Consumer for a websocket-based interface.
// It is expected to be used in real-world scenarios.
type Consumer struct {

	// BindIP indicates the ip for the websokcet listener to bind to,
	// defaults to 0.0.0.0
	BindIP string

	// Port indicates the port to be listening for websocket connection.
	Port int

	// IOPath indicates the url path for exposing websocket for IO,
	// defaults to /io if not set.
	IOPath string

	// CompleterPath indicates the url path for exposing websocket for completer,
	// defaults to /completer if not set
	CompleterPath string

	mux *http.ServeMux

	ioConn *websocket.Conn

	completerConn *websocket.Conn
}

// Init initiates the Consumer for consumption, it must be called before calling
// clui.Connect
func (c *Consumer) Init() (err error) {
	if c.BindIP == "" {
		c.BindIP = "0.0.0.01"
	}
	if c.IOPath == "" {
		c.IOPath = "/io"
	}
	if c.CompleterPath == "" {
		c.CompleterPath = "/completer"
	}
	if c.Port == 0 {
		return errors.New("Port must be set")
	}

	c.mux.HandleFunc(c.CompleterPath, c.handleCompleter)

}

func (c *Consumer) handleCompleter(w http.ResponseWriter, r *http.Request) {

}

// Handle implements the clui.CompletionInfoHandler interface
func (c *Consumer) Handle(ci *protoclui.CompletionInfo) {
}

// Dir implements the clui.Consumer interface
func (c *Consumer) Dir() string {
	panic("not implemented") // TODO: Implement
}

// Input implements the clui.Consumer interface
func (c *Consumer) Input() io.Reader {
	panic("not implemented") // TODO: Implement
}

// Output implements the clui.Consumer interface
func (c *Consumer) Output() io.Writer {
	panic("not implemented") // TODO: Implement
}

// CompOptHandler implements the clui.Consumer interface
func (c *Consumer) CompOptHandler() clui.CompletionInfoHandler {
	panic("not implemented") // TODO: Implement
}

// OnStart implements the clui.Consumer interface
func (c *Consumer) OnStart() {
	panic("not implemented") // TODO: Implement
}
