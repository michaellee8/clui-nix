package wsconsumer

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
	protoclui "github.com/michaellee8/clui-nix/backend/go/pkg/proto/clui"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	completerMut sync.Mutex

	ioMut sync.Mutex

	upgrader websocket.Upgrader
}

// Read implements io.Reader for terminal io
func (c *Consumer) Read(p []byte) (n int, err error) {
	panic("not implemented") // TODO: Implement
}

// Write implements io.Writer for terminal io
func (c *Consumer) Write(p []byte) (n int, err error) {
	panic("not implemented") // TODO: Implement
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

	// Prevent conflict when accessed by multiple clients simulatenously

	c.completerMut.Lock()

	defer c.completerMut.Unlock()

	if c.completerConn != nil {
		// drop the connection if there are already a connection
		logrus.Infof("non-first ws connection to completer attempted by %s", r.RemoteAddr)
		w.WriteHeader(http.StatusConflict)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Infof("unable to upgrade websocket connection: ", errors.Wrap(err, "completer ws upgrade"))
		w.WriteHeader(http.StatusUpgradeRequired)
		return
	}

	c.completerConn = conn

	conn.SetCloseHandler(func(code int, text string) error {
		logrus.Infof("completer: received close message from %s", conn.RemoteAddr)
		message := websocket.FormatCloseMessage(code, "")
		c.completerConn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		c.resetCompleter()
		return nil
	})

}

func (c *Consumer) resetCompleter() {

	// Here we need to deals with both graceful shutdown on close based on error
	// handling, so we just remove the current connection to allow another client
	// to connect

	c.completerMut.Lock()

	defer c.completerMut.Unlock()

	logrus.Infof("completer: resetting connection from %s", c.completerConn.RemoteAddr)

	c.completerConn = nil
}

// Handle implements the clui.CompletionInfoHandler interface
func (c *Consumer) Handle(ci *protoclui.CompletionInfo) {
	logrus.Trace("handling completion info")
	if c.completerConn == nil {
		logrus.Info("assert failed: c.completerConn should be non-nil when handling a completion")
		return
	}
	rb, err := proto.Marshal(ci)
	if err != nil {
		logrus.Error(errors.Wrap(err, "cannot marshal completion info"))
		return
	}
	err = c.completerConn.WriteMessage(websocket.BinaryMessage, rb)
	if err != nil {
		logrus.Error(errors.Wrap(err, "cannot write raw completion info, resetting completerConn"))
		c.resetCompleter()
		return
	}
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
