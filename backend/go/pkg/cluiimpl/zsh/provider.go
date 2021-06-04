package zsh

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rjeczalik/notify"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
	"github.com/spf13/viper"
)

var defaultTranslator = &translator{
	fieldSep: string([]byte{0x00, 0x1b}),       // \0\e
	endSep:   string([]byte{0x00, 0x07, 0x1b}), // \0\a\e
}

var keyListenerOutputEnvKey = "KEY_LISTENER_OUTPUT"

// Provider provides the zsh implementation of clui
type Provider struct {
	dir            string
	input          io.Reader
	output         io.Writer
	compOptHandler clui.CompletionInfoHandler
	comp           *completer
	trans          *translator
	pipeBuffer     string
	installerPath  string
	zshPath        string
	tmpPath        string
	// pipe file opened for read
	pf       *os.File
	pipePath string
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

	// created the named pipe used for communication
	if err := os.MkdirAll(p.tmpPath, 0777); err != nil {
		return errors.Wrap(err, "cannot make tmp dir")
	}
	// no chance of pipeName collision
	pipeName := strconv.Itoa(int(time.Now().UnixNano()))
	pipeName += strconv.Itoa(rand.Int())

	pipePath := filepath.Join(p.tmpPath, pipeName)

	if err := unix.Mkfifo(pipePath, 0666); err != nil {
		return errors.Wrap(err, "cannot create pipe for key listener")
	}

	p.pf, err = os.Open(pipePath)
	if err != nil {
		return errors.Wrap(err, "cannot open the pipe file that had just been created")
	}

	p.pipePath = pipePath

	zdotdir := filepath.Dir(p.zshPath)

	env := os.Environ()
	env = append(env, fmt.Sprintf("ZDOTDIR=%s", zdotdir))
	env = append(env, fmt.Sprintf("%s=%s", keyListenerOutputEnvKey, pipePath))

	cmd := exec.Cmd{
		Path: p.zshPath,
		// force interactive shell here, so maybe we don't need to use pty
		Args:   []string{p.zshPath, "-i"},
		Dir:    p.dir,
		Env:    env,
		Stdin:  p.input,
		Stdout: p.output,
		Stderr: p.output,
	}

	quitListener := make(chan struct{}, 1)

	defer func() { quitListener <- struct{}{} }()

	c := make(chan notify.EventInfo, 1)

	// We do everything that can throw an error in Start so we don't need to
	// deal with error handling in startKeyListener

	if err := notify.Watch(p.pipePath, c, notify.Write, notify.Remove); err != nil {
		return errors.Wrap(err, "cannot setup pipe watch")
	}

	go p.startKeyListener(quitListener, c)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "cannot start zsh")
	}

	return
}

func (p *Provider) startKeyListener(quit chan struct{}, c chan notify.EventInfo) {

	defer notify.Stop(c)
	for {
		select {
		case ei := <-c:
			switch ei.Event() {
			case notify.Write:
				// should not use goroutine here since the current implementation
				// has an not thread-safe buffer. We may be able to remove the
				// need of buffer if we can verify that the whole raw string
				// is always received as a whole
				p.receiveRawCompletionSourceInfo()
			case notify.Remove:
				return
			}
		case <-quit:
			return
		}
	}
}

func (p *Provider) receiveRawCompletionSourceInfo() {
	rb, err := io.ReadAll(p.pf)
	if err != nil {
		// if there is a read error we just discard this trial
		// but we still log it for further debugging anyway
		logrus.Errorf("unable to read pipeFile: %+v", errors.Wrap(err, "cannot read pipeFile"))
		return
	}
	rs := string(rb)
	// we can not ensure that the console will write the whole raw CSI into
	// the fifo instead of splitting it into a few chunks, so we need a buffer
	// here, and then do futher processing only when we can see our endSep at
	// least two times. After that truncate the part of buffer that has already
	// been processed
	// TODO: check if this code can be removed
	p.pipeBuffer += rs
	if strings.Count(p.pipeBuffer, p.trans.endSep) < 2 {
		// nothing we can do here, let's skip to next input
		logrus.Debugf("it is actually possible that the whole string is not written as a whole, current string is %s", p.pipeBuffer)
		return
	}
	pbsp := strings.SplitN(p.pipeBuffer, p.trans.endSep, 3)
	if len(pbsp) != 3 {
		// shouldn't have been possible
		logrus.Errorf("assert error: pbsp should have length of 3, error: %+v", errors.New("assert error"))
		return
	}
	rawcsi := pbsp[1]
	p.pipeBuffer = pbsp[2]

	csi, err := p.trans.translate(rawcsi)
	if err != nil {
		logrus.Errorf("cannot translate raw CSI:%+v", errors.Wrap(err, "cannot translate raw CSI"))
		return
	}
	ci, err := p.comp.getCompletion(csi)
	if err != nil {
		logrus.Errorf("cannot get cmpletion: %+v", errors.Wrap(err, "cannot get completion"))
	}
	p.compOptHandler.Handle(ci)
}

type translator struct {
	fieldSep string
	endSep   string
}

func (t *translator) translate(ris string) (csi completionSourceInfo, err error) {
	// format: $line;$col|$pwd|$LBUFFER|$RBUFFER|$BUFFER\n
	// ; is as is, | is fieldsep, \n is endsep, endsep can be differeniated with
	// fieldsep with the middle \a bell character
	// ris := string(ri)
	spris := strings.Split(ris, t.fieldSep)
	if len(spris) != 5 {
		return csi, errors.New("raw completion source info translate error: number of field is not exactly 5")
	}

	sppos := strings.Split(spris[0], ";")
	if len(sppos) != 2 {
		return csi, errors.New("raw completion source info translate error: number of field is not exactly 2")
	}

	csi.line, err = strconv.Atoi(sppos[0])
	if err != nil {
		return
	}

	csi.col, err = strconv.Atoi(sppos[1])
	if err != nil {
		return
	}

	csi.dir = spris[1]

	csi.lbuffer = spris[2]

	csi.rbuffer = spris[3]

	csi.buffer = spris[4]

	return
}
