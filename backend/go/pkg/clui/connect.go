package clui

import "github.com/pkg/errors"

// Connect takes a Provider and a Consumer, perform the connection logic between
// them. and then starts the Provider. It returns an error if there are any error
// in the process, otherwise it returns nil.
func Connect(p Provider, c Consumer) (err error) {
	p.SetDir(c.Dir())
	p.SetInput(c.Input())
	p.SetOutput(c.Output())
	p.SetCompOptHandler(c.CompOptHandler())
	go c.OnStart()

	return errors.Wrap(p.Start(), "clui connect failed")

}
