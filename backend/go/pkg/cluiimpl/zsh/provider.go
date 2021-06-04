package zsh

import (
	"io"

	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
)

type Provider struct {
	dir            string
	input          io.Reader
	output         io.Writer
	compOptHandler clui.StringSliceHandler
}

func (p *Provider) SetDir(s string) {
	p.dir = s
}

func (p *Provider) SetInput(r io.Reader) {
	p.input = r
}

func (p *Provider) SetOutput(w io.Writer) {
	p.output = w
}

func (p *Provider) SetCompOptHandler(j clui.StringSliceHandler) {
	p.compOptHandler = j
}

func NewProvider() *Provider {
	return &Provider{}
}
