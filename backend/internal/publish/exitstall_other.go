//go:build !windows

package publish

func (c *child) killOnStalledExit(waited <-chan struct{}, tag string) {}
