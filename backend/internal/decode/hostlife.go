package decode

import (
	"fmt"
	"os"

	"bjoernblessin.de/go-utils/util/logger"
)

// What the host reports back, and what the backend does when it stops answering.

// hostGone drops a host whose control connection failed, so the next open spawns another.
//
// The process gets killed even where its socket merely ran out of time.
// A host wedged in a display driver holds the GPU and every decode on it, and nothing else can reach it.
// The report runs on its own goroutine because the caller may hold a lock that the callbacks take.
func (c *Client) hostGone(err error) {
	c.mu.Lock()
	ctrl, proc := c.ctrl, c.proc
	if ctrl == nil {
		c.mu.Unlock()
		return
	}
	ended := c.events
	c.ctrl, c.proc, c.socket, c.events = nil, nil, "", map[ID]Events{}
	if proc != nil {
		proc.dropped = true
	}
	c.mu.Unlock()

	ctrl.Close()
	logger.Warnf("the decode host stopped answering: %v", err)

	logPath := ""
	if proc != nil {
		proc.cmd.Process.Kill()
		logPath = proc.logPath
	}
	go reportEnded(ended, logPath)
}

// hostExited reports every decode the host was running.
//
// A GPU reset aborts the host where it stands, so this is an ordinary way for a decode to end
// rather than an exceptional one, and each is reported the way a pipeline that stopped by itself
// is.
func (c *Client) hostExited(proc *process) {
	// A host that exited answers no dial, and the next spawn makes its own directory.
	defer os.RemoveAll(proc.dir)

	c.mu.Lock()
	if proc.dropped {
		// hostGone cleared this host's state, and its successor may be running by now.
		c.mu.Unlock()
		return
	}
	ended := c.events
	c.events = map[ID]Events{}
	closed := c.closed
	c.ctrl, c.proc, c.socket = nil, nil, ""
	c.mu.Unlock()

	if !closed {
		reportEnded(ended, proc.logPath)
	}
}

// reportEnded tells each decode that its host went.
// The log is named because it holds what the host said on its way down, which is the one place
// the driver's own message lands.
func reportEnded(ended map[ID]Events, logPath string) {
	if len(ended) == 0 {
		return
	}
	logger.Warnf("the decode host exited with %d decode(s) running, its log is at %s", len(ended), logPath)
	for id, ev := range ended {
		logger.Debugf("the decode of %s ended with its host", id)
		if ev.OnEnd != nil {
			ev.OnEnd(fmt.Sprintf("the decoding process stopped, its log is at %s", logPath))
		}
	}
}

// readLifecycle delivers what decodes report unasked, until the host goes.
//
// A message about a decode this side has forgotten is dropped: a stop takes the callbacks away
// first, so an end announced afterwards would report a decode the caller closed.
func (c *Client) readLifecycle(conn *conn) {
	defer conn.Close()

	for {
		var msg lifecycleMessage
		if err := conn.recv(&msg); err != nil {
			return
		}

		c.mu.Lock()
		ev, present := c.events[msg.ID]
		c.mu.Unlock()

		if !present {
			continue
		}
		switch msg.Kind {
		case lifeLive:
			if ev.OnLive != nil {
				ev.OnLive()
			}
		case lifeEnd:
			if ev.OnEnd != nil {
				ev.OnEnd(msg.Message)
			}
		}
	}
}
