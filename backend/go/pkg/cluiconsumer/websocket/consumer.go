package wsconsumer

import (
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
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

	ser *http.Server

	ioConn *websocket.Conn

	completerConn *websocket.Conn

	completerMut sync.Mutex

	ioMut sync.RWMutex

	upgrader websocket.Upgrader

	ioReadWait  chan struct{}
	ioWriteWait chan struct{}

	ioConnReader io.Reader
}

// Read implements io.Reader for terminal io
func (c *Consumer) Read(p []byte) (n int, err error) {
	c.ioMut.RLock()
	if c.ioConn == nil {
		// wait for an connection
		<-c.ioReadWait
		if c.ioConn == nil {
			// if this assertion failed, next steps will be a null pointer panic anyway, so we do the panic ourselves
			logrus.Panic("fatal: assert failed: c.ioConn must be populated when c.ioReadWait is received, terminating program.")
			return 0, errors.New("fatal: assert failed: c.ioConn is still nil")
		}
	}
	c.ioMut.RUnlock()

	// reference on converting ReadMessage into Read
	// https://github.com/gorilla/websocket/issues/282

	for {
		if c.ioConnReader == nil {
			// Advance to next message.
			var err error

			var mt int
			mt, c.ioConnReader, err = c.ioConn.NextReader()
			if err != nil {
				return 0, errors.Wrap(err, "ws consumer read: ")
			}
			if mt != websocket.BinaryMessage {
				logrus.Infof("assert failed: mt must be BinaryMessage, got %d instead", mt)
			}

		}
		n, err := c.ioConnReader.Read(p)
		if err == io.EOF {
			// At end of message.
			c.ioConnReader = nil
			if n > 0 {
				return n, nil
			} else {
				// No data read, continue to next message
				continue
			}
		}
		return n, errors.Wrap(err, "ws consumer read: cannot read message: ")
	}

}

// Write implements io.Writer for terminal io
func (c *Consumer) Write(p []byte) (n int, err error) {
	c.ioMut.RLock()
	if c.ioConn == nil {
		// wait for an connection
		<-c.ioWriteWait
		if c.ioConn == nil {
			logrus.Panic("fatal: assert failed: c.ioConn must be populated when c.ioWriteWait is received, terminating program.")
			return 0, errors.New("fatal: assert failed: c.ioConn is still nil")
		}
	}
	c.ioMut.RUnlock()

	err = c.ioConn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Init initiates the Consumer for consumption, it must be called before calling
// clui.Connect
func (c *Consumer) Init() (err error) {
	if c.BindIP == "" {
		c.BindIP = "0.0.0.0"
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

	c.mux = http.NewServeMux()

	c.mux.HandleFunc(c.CompleterPath, c.handleCompleter)
	c.mux.HandleFunc(c.IOPath, c.handleIO)

	c.ser = &http.Server{
		Addr:    net.JoinHostPort(c.BindIP, strconv.Itoa(c.Port)),
		Handler: c.mux,
	}

	go func() {
		if err := c.ser.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				logrus.Infof("ws consumer server closed or shutdown: %+v", errors.Wrap(err, "server shutdown or closed"))
			} else {
				logrus.Fatalf("ws consumer server cannot start: %+v", errors.Wrap(err, "server cannot start"))
			}
		}
	}()

	return

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

func (c *Consumer) handleIO(w http.ResponseWriter, r *http.Request) {

	// Prevent conflict when accessed by multiple clients simulatenously

	c.ioMut.Lock()

	defer c.ioMut.Unlock()

	if c.ioConn != nil {
		// drop the connection if there are already a connection
		logrus.Infof("non-first ws connection to io attempted by %s", r.RemoteAddr)
		w.WriteHeader(http.StatusConflict)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Infof("unable to upgrade websocket connection: ", errors.Wrap(err, "io ws upgrade"))
		w.WriteHeader(http.StatusUpgradeRequired)
		return
	}

	c.ioConn = conn

	conn.SetCloseHandler(func(code int, text string) error {
		logrus.Infof("io: received close message from %s", conn.RemoteAddr)
		message := websocket.FormatCloseMessage(code, "")
		c.ioConn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		c.resetCompleter()
		return nil
	})

	c.ioReadWait <- struct{}{}
	c.ioWriteWait <- struct{}{}
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
func (c *Consumer) resetIO() {

	// Here we need to deals with both graceful shutdown on close based on error
	// handling, so we just remove the current connection to allow another client
	// to connect

	c.ioMut.Lock()

	defer c.ioMut.Unlock()

	logrus.Infof("io: resetting connection from %s", c.ioConn.RemoteAddr)

	c.ioConn = nil
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
	return os.Getenv("HOME")
}

// Input implements the clui.Consumer interface
func (c *Consumer) Input() io.Reader {
	return c
}

// Output implements the clui.Consumer interface
func (c *Consumer) Output() io.Writer {
	return c
}

// CompOptHandler implements the clui.Consumer interface
func (c *Consumer) CompOptHandler() clui.CompletionInfoHandler {
	return c
}

// OnStart implements the clui.Consumer interface
func (c *Consumer) OnStart() {
}
