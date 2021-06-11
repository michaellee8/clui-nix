package zsh

import (
	"fmt"
	"github.com/kr/pty"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	protoclui "github.com/michaellee8/clui-nix/backend/go/pkg/proto/clui"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
	"github.com/spf13/viper"
)

var keyListenerOutputEnvKey = "KEY_LISTENER_OUTPUT"

// Provider provides the zsh implementation of clui
type Provider struct {
	dir            string
	input          io.Reader
	output         io.Writer
	compOptHandler clui.CompletionInfoHandler
	winsizeChan    chan pty.Winsize
	comp           *completer
	trans          *translator
	pipeBuffer     string
	installerPath  string
	zshPath        string
	tmpPath        string
	// pipe file opened for read
	pf net.Listener
	// it is a socket now, pipe is just here for historial reaons
	// TODO: rename pipe* to sock*
	pipePath string
}

func (p *Provider) SetWinsizeChan(winsizes chan pty.Winsize) {
	p.winsizeChan = winsizes
}

// SetDir sets the current working directory of the process
func (p *Provider) SetDir(s string) {
	p.dir = s
}

// SetInput sets the input stream used for Stdin
func (p *Provider) SetInput(r io.Reader) {
	p.input = r
}

// SetOutput sets the output stream used for both Stdout and Stderr
func (p *Provider) SetOutput(w io.Writer) {
	p.output = w
}

// SetCompOptHandler sets the completion option handler
func (p *Provider) SetCompOptHandler(j clui.CompletionInfoHandler) {
	p.compOptHandler = j
}

// NewProvider returns a new instance of Provider using default options
func NewProvider() *Provider {
	var defaultTranslator = &translator{
		fieldSep: string([]byte{0x00, 0x1b}),       // \0\e
		endSep:   string([]byte{0x00, 0x07, 0x1b}), // \0\a\e
	}
	var defaultCompleter = &completer{
		completerScriptPath: viper.GetString("ZSH_COMPLETER_SCRIPT_PATH"),
		zshPath:             viper.GetString("ZSH_PATH"),
		maxHelp:             10,
	}
	return &Provider{
		comp:          defaultCompleter,
		trans:         defaultTranslator,
		installerPath: viper.GetString("ZSH_COMPLETER_SCRIPT_PATH"),
		zshPath:       viper.GetString("ZSH_PATH"),
		tmpPath:       viper.GetString("CLUI_TMP_PATH"),
	}
}

// Start starts performs the required preparation and then starts the zsh
// process, as well as start providing completion results via compOptHandler
func (p *Provider) Start() (err error) {

	// validate we have correct options set by clui first
	if p.dir == "" {
		return errors.New("zsh provider: dir is not set")
	}
	if p.input == nil {
		return errors.New("zsh provider: input is not set")
	}
	if p.output == nil {
		return errors.New("zsh provider: output is not set")
	}
	if p.compOptHandler == nil {
		return errors.New("zsh provider: compOptHandler is not set")
	}
	if p.winsizeChan == nil {
		return errors.New("zsh provider: winsizeChan is not set")
	}

	// created the named pipe used for communication
	if err := os.MkdirAll(p.tmpPath, 0777); err != nil {
		return errors.Wrap(err, "cannot make tmp dir")
	}
	// no chance of pipeName collision
	pipeName := strconv.Itoa(int(time.Now().UnixNano()))
	pipeName += strconv.Itoa(rand.Int())

	sockPath := filepath.Join(p.tmpPath, pipeName)

	if p.pf, err = net.Listen("unixpacket", sockPath); err != nil {
		return errors.Wrap(err, "cannot create unixpacket socket for key listener")
	}

	defer func() {
		if err := p.pf.Close(); err != nil {
			logrus.Errorln(errors.Wrap(err, "closing key listener socket failed"))
		}
	}()

	p.pipePath = sockPath

	zdotdir := filepath.Dir(p.installerPath)

	env := os.Environ()
	env = append(env, fmt.Sprintf("ZDOTDIR=%s", zdotdir))
	env = append(env, fmt.Sprintf("%s=unixpacket://%s", keyListenerOutputEnvKey, sockPath))

	cmd := exec.Cmd{
		Path: p.zshPath,
		// force interactive shell here, so maybe we don't need to use pty
		Args: []string{p.zshPath, "-i"},
		Dir:  p.dir,
		Env:  env,
	}

	go p.startKeyListener()

	ptmx, err := pty.Start(&cmd)

	if err != nil {
		logrus.Error("cannot start zsh: ", err)
		return errors.Wrap(err, "cannot start zsh")
	}

	defer func() {
		if err = ptmx.Close(); err != nil {
			logrus.Error("cannot close zsh: ", err)

		}
	}()

	go func() {
		if _, err = io.Copy(ptmx, p.input); err != nil {
			logrus.Error("cannot copy ptmx stdout to p.input: ", err)
		}
	}()

	go func() {
		for winsize := range p.winsizeChan {
			if err := pty.Setsize(ptmx, &winsize); err != nil {
				logrus.Error("zsh provder: unable to resize pty: ", err)
			}
		}
	}()

	if _, err = io.Copy(p.output, ptmx); err != nil {
		logrus.Error("cannot copy p.output to ptmx stdin: ", err)
		return errors.Wrap(err, "cannot copy")
	}

	return
}

func (p *Provider) startKeyListener() {

	logrus.Trace("starting key listener")

	for {
		conn, err := p.pf.Accept()
		if err != nil {
			logrus.Errorln(errors.Wrap(err, "key listener accept failed"))
			continue
		}
		go p.receiveRawCompletionSourceInfo(conn)
	}

}

func (p *Provider) receiveRawCompletionSourceInfo(conn net.Conn) {
	logrus.Trace("receiving raw CSI")
	rcsi, err := io.ReadAll(conn)
	if err != nil {
		// if there is a read error we just discard this trial
		// but we still log it for further debugging anyway
		logrus.Errorf("unable to read pipeFile: %+v", errors.Wrap(err, "cannot read pipeFile"))
		return
	}

	if err := conn.Close(); err != nil {
		logrus.Error(errors.Wrap(err, "cannot close conn"))
	}

	csi, err := p.trans.translate(rcsi)
	if err != nil {
		logrus.Errorf("cannot translate raw CSI: %+v", errors.Wrap(err, "cannot translate raw CSI"))
		return
	}
	ci, err := p.comp.getCompletion(csi)
	if err != nil {
		logrus.Errorf("cannot get completion: %+v, %+v", errors.Wrap(err, "cannot get completion"), err)
	}
	p.compOptHandler.Handle(&ci)
}

type translator struct {
	fieldSep string
	endSep   string
}

func (t *translator) translate(rcsi []byte) (csi completionSourceInfo, err error) {

	pcsi := protoclui.CompletionSourceInfo{}

	if err := proto.Unmarshal(rcsi, &pcsi); err != nil {
		return completionSourceInfo{}, errors.Wrap(err, "cannot unmarshal raw CSI")
	}

	csi.line = int(pcsi.Line)
	csi.col = int(pcsi.Col)
	csi.dir = pcsi.Dir

	csi.lbuffer = pcsi.LBuffer

	csi.rbuffer = pcsi.RBuffer

	csi.buffer = pcsi.Buffer

	return
}
