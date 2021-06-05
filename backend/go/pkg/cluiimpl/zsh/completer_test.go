package zsh

import (
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var testCompleter *completer
var testDir = "~/.local/tmp/completer_test"

func getCompleterScriptPath() string {

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalln("unable to get pwd, exiting")
	}
	log.Println("Running with pwd as: ", pwd)
	scriptPath := pwd                     // zsh
	scriptPath = filepath.Dir(scriptPath) // cluiimpl
	scriptPath = filepath.Dir(scriptPath) // pkg
	scriptPath = filepath.Dir(scriptPath) // go
	scriptPath = filepath.Dir(scriptPath) // backend
	scriptPath = filepath.Join(scriptPath, "scripts", "capture.zsh")
	return scriptPath
}

func TestMain(m *testing.M) {
	if err := os.MkdirAll(testDir, 0777); err != nil {
		log.Fatalln("unable to create tmpdir, exiting")
	}
	testCompleter = &completer{zshPath: "/bin/zsh", completerScriptPath: getCompleterScriptPath(), maxHelp: 10}
	logrus.SetLevel(logrus.DebugLevel)
	os.Exit(m.Run())
}

func TestCompletion(t *testing.T) {
	require := require.New(t)
	csi := completionSourceInfo{
		dir:     testDir,
		col:     15,
		line:    20,
		lbuffer: "vi",
		rbuffer: "",
		buffer:  "vi",
	}

	ci, err := testCompleter.getCompletion(csi)

	if err != nil {
		if err, ok := err.(*fs.PathError); ok {
			t.Errorf("getCompletion error: %v", err.Unwrap())
		}
	}

	require.Nil(err)

	require.Equal(ci.Col, int64(csi.col))
	require.Equal(ci.Line, int64(csi.line))
	require.Equal(ci.IsEmpty, false)
	require.Equal(ci.IsFirst, true)
	require.Equal(ci.BufferLength, int64(2))

}

func TestWordCount(t *testing.T) {
	require := require.New(t)

	require.Equal((&completionSourceInfo{buffer: "   "}).isEmpty(), true)
	require.Equal((&completionSourceInfo{buffer: "  \t\t\t "}).isEmpty(), true)
	require.Equal((&completionSourceInfo{buffer: "\t word  \t "}).isFirstWord(), true)
	require.Equal((&completionSourceInfo{buffer: "\t word  \t word2 \t  \t  "}).countWord(), int64(2))
}

// BenchmarkCompletion benchmark the completion speed of a random 1 letter command
// suffix
func BenchmarkCompletion(b *testing.B) {
	logrus.SetLevel(logrus.ErrorLevel)
	cmdLength := 1
	var cmdb []byte
	for i := 0; i < cmdLength; i++ {
		cmdb = append(cmdb, byte(97+rand.Intn(26)))
	}

	randCmd := string(cmdb)
	// TODO: use actual random comand, currently random command will stall the
	// benchmark indefinitely
	randCmd = "vi"
	b.Logf("running with command %s", randCmd)

	benchCompleter := &completer{zshPath: "/bin/zsh", completerScriptPath: getCompleterScriptPath(), maxHelp: b.N}
	csi := completionSourceInfo{
		dir:     testDir,
		col:     15,
		line:    20,
		lbuffer: randCmd,
		rbuffer: "",
		buffer:  randCmd,
	}

	b.ResetTimer()

	_, err := benchCompleter.getCompletion(csi)

	if err != nil {
		b.Fatal("cannot get completion, stopping: ", err)
	}

}
