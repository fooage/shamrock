package raft

import (
	"errors"
	"net"
	"time"

	"go.uber.org/zap"
)

// stoppableListener sets TCP keep-alive timeouts on accepted
// connections and waits on stopCh message.
type stoppableListener struct {
	*net.TCPListener
	stopCh <-chan struct{}
	logger *zap.Logger
}

func newStoppableListener(logger *zap.Logger, addr string, stopCh <-chan struct{}) (*stoppableListener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("net.Listen tcp addr error", zap.String("addr", addr))
		return nil, err
	}
	return &stoppableListener{listener.(*net.TCPListener), stopCh, logger}, nil
}

// Accept implement the method defined by the net.Listener interface. After the
// connection is established, it will listen to the application layer stop
// message to stop the Raft service.
func (ln stoppableListener) Accept() (net.Conn, error) {
	connectCh := make(chan *net.TCPConn, 1)
	errorCh := make(chan error, 1)

	go func() {
		defer func() {
			// catch panic error avoid this node crashed
			if err := recover(); err != nil {
				ln.logger.Error("panic when accept tcp connect", zap.Error(err.(error)))
			}
		}()
		connect, err := ln.AcceptTCP()
		if err != nil {
			errorCh <- err
			return
		}
		connectCh <- connect
	}()

	// synchronously select stop channel and connection
	select {
	case <-ln.stopCh:
		return nil, errors.New("tcp listener will be stopped")
	case err := <-errorCh:
		return nil, err
	case connect := <-connectCh:
		_ = connect.SetKeepAlive(true)
		_ = connect.SetKeepAlivePeriod(3 * time.Minute)
		return connect, nil
	}
}
