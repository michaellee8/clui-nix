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
