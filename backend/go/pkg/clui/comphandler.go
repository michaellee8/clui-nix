package clui

import (
	protoclui "github.com/michaellee8/clui-nix/backend/go/pkg/proto/clui"
)

type StringSliceHandler interface {
	Handle(p []string)
}

type CompletionInfoHandler interface {
	Handle(ci *protoclui.CompletionInfo)
}
