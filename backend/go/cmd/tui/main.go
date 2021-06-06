package main

import (
	"log"

	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
	"github.com/michaellee8/clui-nix/backend/go/pkg/cluiconsumer/tui"
	"github.com/michaellee8/clui-nix/backend/go/pkg/cluiimpl/zsh"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()

	viper.SetDefault(
		"ZSH_COMPLETER_SCRIPT_PATH",
		"/home/michaellee8/personal-projects/clui-nix/backend/scripts/capture.zsh",
	)
	viper.SetDefault(
		"ZSH_PATH",
		"/bin/zsh",
	)
	viper.SetDefault(
		"CLUI_TMP_PATH",
		"/tmp/clui-tui",
	)
	viper.SetDefault(
		"GOLOG",
		"fatal",
	)

	logLevel, err := logrus.ParseLevel(viper.GetString("GOLOG"))
	if err != nil {
		logrus.Fatalln(errors.Wrap(err, "cannot parse log level"))
		return
	}
	logrus.SetLevel(logLevel)

	zshProvider := zsh.NewProvider()
	tuiConsumer := tui.Consumer{}
	if err := tuiConsumer.Init(); err != nil {
		log.Fatalln("cannot init tuiconsumer: ", err)
	}

	if err := clui.Connect(zshProvider, &tuiConsumer); err != nil {
		log.Fatalln("cannot connect: ", err)
	}
}
