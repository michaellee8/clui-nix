package zsh

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	protoclui "github.com/michaellee8/clui-nix/backend/go/pkg/proto/clui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type completionSourceInfo struct {
	col     int
	line    int
	dir     string
	lbuffer string
	rbuffer string
	buffer  string
}

func (csi *completionSourceInfo) words() []string {
	return strings.Fields(csi.buffer)
}

func (csi *completionSourceInfo) countWord() int64 {

	sepbuf := csi.words()

	// Count the number of non-empty word in sepbuf
	count := 0
	for _, s := range sepbuf {
		if s != "" {
			count++
		}
	}

	return int64(count)
}

// isFirstWord returns whether the we are completing for the first word, which
// is in most cases actual command
func (csi *completionSourceInfo) isFirstWord() bool {

	return csi.countWord() == 1

}

// isEmpty returns whether the we are completing for no word, which means the
// user has not typed any command yet and we can suggest some by ourselves
func (csi *completionSourceInfo) isEmpty() bool {

	return csi.countWord() == 0

}

type completer struct {
	completerScriptPath string
	zshPath             string
	// maxHelp indicates the number of first compopt we should provide description for
	maxHelp int
}

var defaultCompleter = &completer{
	completerScriptPath: viper.GetString("ZSH_COMPLETER_SCRIPT_PATH"),
	zshPath:             viper.GetString("ZSH_PATG"),
	maxHelp:             10,
}

// Do not process these commands, those are known to be buggy
var blacklistedCommands = []string{
	"vimtutor",
}

// getCompletion provide the hacky logic the retrieve the completions results
func (co *completer) getCompletion(csi completionSourceInfo) (ci protoclui.CompletionInfo, err error) {

	// Obtain Completion Results
	cmd := exec.Cmd{
		Path: co.zshPath,
		Args: []string{co.zshPath, "-c", fmt.Sprintf("%s '%s'", co.completerScriptPath, csi.buffer)},
		Dir:  csi.dir,
		Env:  nil,
	}
	out, err := cmd.Output()
	if err != nil {
		return
	}
	outStr := string(out)

	cts := strings.Split(outStr, "\r\n")

	// Compile these completions results into our CompletionInfo

	ci.Col = int64(csi.col)
	ci.Line = int64(csi.line)
	ci.IsEmpty = csi.isEmpty()
	ci.IsFirst = csi.isFirstWord()
	ci.BufferLength = int64(len(csi.buffer))

	if ci.IsEmpty {
		// we should suggest our own completion results if there are no command
		// has already be input
		return
	}

COMPOPT_LOOP:
	for compoptI, compopt := range cts {
		// Since our script is hacky, skip empty results
		if compopt == "" {
			continue
		}
		// Skip blacklisted commands
		for _, bcmd := range blacklistedCommands {
			if bcmd == compopt {
				logrus.Debug("ignored blacklisted command: ", compopt)
				continue COMPOPT_LOOP
			}
		}
		logrus.Debug("compopt: ", compopt)
		var description string

		// if this is the first word, we can provide description of the command
		// by taking the first line of <cmd> --help
		// TODO: cache the help results
		// TODO: preload the help results for common commands that exist in the
		//		 cotainer enviroment into the clinet
		if ci.IsFirst && (co.maxHelp == 0 || (co.maxHelp > 0 && compoptI < co.maxHelp)) {

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			// disable zsh for faster help results
			// helpCmd := exec.CommandContext(ctx, co.zshPath, "-c", fmt.Sprintf("%s --help", compopt))
			cmdpath, err := exec.LookPath(compopt)
			if err == nil {

				helpCmd := exec.CommandContext(ctx, cmdpath, "--help")

				combout, _ := helpCmd.CombinedOutput()

				// we can ignore herr if it is a ExitError, otherwise we will have
				// to return error
				// switch e := herr.(type) {
				// case *exec.ExitError:
				// default:
				// 	// or we can just ignore any kind of error anyway, it shouldn't
				// 	// be important
				// 	err = e
				// 	return
				// }

				// get the first line of combout
				description = strings.Split(string(combout), "\n")[0]
				logrus.Debug("description: ", description)
			}

		}

		// we will also need to provide the actual input the frontend should
		// type in if they want to accept the suggestion

		var actualInput string
		var shouldInput bool
		lastWord := csi.words()[len(csi.words())-1]
		// normally, lastWord should overlap with compopt, if that is not the
		// case, we just pass the whole compopt as actualInput and then
		// tell our frontend not to complete this word, and just let user type
		// the suggestion instead.

		if compopt[:len(lastWord)] == lastWord {
			actualInput = compopt[len(lastWord):]
			shouldInput = true
		} else {
			actualInput = compopt
			shouldInput = false
		}

		// processing done, now add it to our suggestions
		ci.Entries = append(ci.Entries, &protoclui.CompletionEntry{
			ActualInput: actualInput,
			ShouldInput: shouldInput,
			Description: description,
			Suggestion:  compopt,
			Level:       0,
		})

	}
	return

}
