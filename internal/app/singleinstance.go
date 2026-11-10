package app

import (
	"net"
	"time"
)

// singleInstanceAddr is the loopback rendezvous for single-instance
// detection. Any local process that connects only asks the running
// instance to raise its window; nothing sensitive travels here.
const singleInstanceAddr = "127.0.0.1:41739"

// acquireSingleInstance binds addr when this is the first instance and
// serves raise signals until release is called. When another instance
// already holds addr it is signalled and first returns false, so the
// caller should exit without opening a second window.
//
// Note: raising actually focuses the window on X11; on Wayland the
// compositor ignores programmatic focus, so the user still switches
// with the dock or Alt-Tab — but no duplicate window ever opens.
func acquireSingleInstance(addr string, onRaise func()) (bound string, release func(), first bool) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		signalFirstInstance(addr)
		return "", func() {}, false
	}
	go serveRaise(listener, onRaise)
	return listener.Addr().String(), func() { _ = listener.Close() }, true
}

// signalFirstInstance asks the running instance to raise its window.
// Failure is ignored: either way the caller must not continue.
func signalFirstInstance(addr string) {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, _ = conn.Write([]byte("raise"))
}

func serveRaise(listener net.Listener, onRaise func()) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return // released
		}
		conn.Close()
		onRaise()
	}
}
