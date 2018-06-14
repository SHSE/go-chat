package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

type testContext struct {
	commands    chan command
	connects    chan int
	disconnects chan int
	events      chan string
	unicast     unicast
	nextResult  error
}

func (context *testContext) command(command command) error {
	context.commands <- command
	result := context.nextResult
	context.nextResult = nil
	return result
}

func (context *testContext) connected(clientId int) {
	context.connects <- clientId
}

func (context *testContext) disconnected(clientId int) {
	context.disconnects <- clientId
}

func (context *testContext) waitForEvent(t *testing.T, event string) {
	for event := range context.events {
		if event == event {
			return
		}
	}

	t.Fail()
}

func runTestServer() (stop func(), address string, textCtx *testContext) {
	ctx, cancel := context.WithCancel(context.Background())
	logger, _ := zap.NewDevelopment()
	server := NewServer(logger, prometheus.NewRegistry())
	textCtx = &testContext{
		make(chan command, 100),
		make(chan int, 100),
		make(chan int, 100),
		make(chan string, 100),
		server,
		nil,
	}

	port, err := freeport.GetFreePort()

	if err != nil {
		panic(err)
	}

	address = fmt.Sprintf("localhost:%d", port)

	done := make(chan error, 1)

	go func() {
		done <- server.Run(ctx, address, textCtx)
	}()

	stop = func() {
		cancel()
		<-done
	}

	return
}

func withSession(t *testing.T, action func(*testContext, *Session)) {
	stop, address, handler := runTestServer()
	session, err := NewSession(address, handler.events)

	assert.Nil(t, err)

	action(handler, session)

	session.Close()
	stop()
}

func (session *Session) sendCommandAndExpectOK(t *testing.T, name string, args []string) {
	result, err := session.SendCommand(name, args)
	assert.True(t, result)
	assert.Nil(t, err)
}

func (session *Session) join(t *testing.T, name string) {
	session.sendCommandAndExpectOK(t, "join", []string{name})
}

func TestInvokesConnectMethodWhenNewClientConnected(t *testing.T) {
	withSession(t, func(context *testContext, session *Session) {
		<-context.connects
	})
}

func TestDeliversNotificationToClient(t *testing.T) {
	withSession(t, func(context *testContext, session *Session) {
		clientId := <-context.connects

		session.join(t, "john")

		context.unicast.sendTo(clientId, "hello")

		context.waitForEvent(t, "hello")
	})
}

func TestReturnsErrorWhenCommandFailed(t *testing.T) {
	withSession(t, func(context *testContext, session *Session) {
		context.nextResult = errors.New("boom")

		result, err := session.SendCommand("join", []string{"john"})

		assert.Nil(t, err)
		assert.False(t, result)
	})
}

func TestInvokesDisconnectMethodWhenNewClientDisconnected(t *testing.T) {
	withSession(t, func(ctx *testContext, session *Session) {
		session.Close()
		assert.Equal(t, <-ctx.connects, <-ctx.disconnects)
	})
}
