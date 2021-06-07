package main

import (
	"github.com/michaellee8/clui-nix/backend/go/pkg/clui"
	wsconsumer "github.com/michaellee8/clui-nix/backend/go/pkg/cluiconsumer/websocket"
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
		"/tmp/ws",
	)
	viper.SetDefault(
		"GOLOG",
		"fatal",
	)
	viper.SetDefault(
		"PORT",
		"8080",
	)
	logLevel, err := logrus.ParseLevel(viper.GetString("GOLOG"))
	if err != nil {
		logrus.Fatalln(errors.Wrap(err, "cannot parse log level"))
		return
	}
	logrus.SetLevel(logLevel)

	zshProvider := zsh.NewProvider()
	wsConsumer := wsconsumer.Consumer{
		Port: viper.GetInt("PORT"),
	}

	if err := wsConsumer.Init(); err != nil {

		logrus.Fatalln("cannot init wsconsumer: ", err)
	}
	if err := clui.Connect(zshProvider, &wsConsumer); err != nil {
		logrus.Fatalln("cannot connect: ", err)
	}
}
