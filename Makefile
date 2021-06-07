all:

proto:
	buf generate -o .

test:
	go test ./...

bench:
	# go test -bench . ./...
	go test -bench . -run=^a ./backend/go/pkg/cluiimpl/zsh

profile:
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench . -run=^a ./backend/go/pkg/cluiimpl/zsh

build: zkeylis

zkeylis:
	go build -o ./backend/scripts/zkeylis ./backend/go/cmd/zkeylis/main.go

runtui:
	go run ./backend/go/cmd/tui

runws:
	go run ./backend/go/cmd/ws
