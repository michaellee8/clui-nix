package zkeylis

import (
	"flag"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"log"

	"github.com/golang/protobuf/proto"
	"github.com/michaellee8/clui-nix/backend/go/pkg/proto/clui"
)

func debugPrintln(s ...interface{}) {
	if os.Getenv("GODEBUG") != "" {
		log.Println(s...)
	}
}

func debugPrintf(f string, s ...interface{}) {
	if os.Getenv("GODEBUG") != "" {
		log.Printf(f, s...)
	}
}

func main() {
	var pos, dir, buffer, lbuffer, rbuffer string
	var urlstr string

	flag.StringVar(&pos, "pos", "", "postion of current cursor in line;col form")
	flag.StringVar(&dir, "dir", "", "current working directory")
	flag.StringVar(&buffer, "buffer", "", "zsh buffer")
	flag.StringVar(&lbuffer, "lbuffer", "", "zsh lbuffer")
	flag.StringVar(&rbuffer, "rbuffer", "", "zsh rbuffer")
	flag.StringVar(&urlstr, "url", "", "url of the listening server")

	if urlstr == "" {
		debugPrintln("url is empty")
		return
	}

	debugPrintf(
		"zkeylis debug: pos: %s, dir: %s, buffer: %s, lbuffer: %s, rbuffer: %s, url: %s\n",
		pos, dir, buffer, lbuffer, rbuffer, urlstr,
	)

	var line, col int

	var err error

	possp := strings.Split(pos, ";")

	if line, err = strconv.Atoi(possp[0]); err != nil {
		debugPrintln(err)
		return
	}

	if col, err = strconv.Atoi(possp[1]); err != nil {
		debugPrintln(err)
		return
	}

	csi := clui.CompletionSourceInfo{
		Line:    int64(line),
		Col:     int64(col),
		Dir:     dir,
		Buffer:  buffer,
		LBuffer: lbuffer,
		RBuffer: rbuffer,
	}

	u, err := url.Parse(urlstr)

	if err != nil {
		debugPrintln(err)
		return
	}

	debugPrintf(
		"url parse result: scheme: %s, host: %s, path: %s\n",
		u.Scheme, u.Host, u.Path,
	)

	var conn net.Conn

	if strings.Contains(u.Scheme, "unix") {
		conn, err = net.Dial(u.Scheme, u.Path)
	} else {
		conn, err = net.Dial(u.Scheme, u.Host)
	}

	if err != nil {
		debugPrintln(err)
		return
	}

	defer conn.Close()

	rawCsi, err := proto.Marshal(csi)

	if err != nil {
		debugPrintln(err)
		return
	}

	_, err = conn.Write(rawCsi)
	if err != nil {
		debugPrintln(err)
		return
	}
}
