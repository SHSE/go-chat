package transport

import (
	"context"
	"errors"
	"fmt"
	"github.com/phayes/freeport"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
	"github.com/stretchr/testify/require"
)

type testContext struct {
	commands    chan Command
	connects    chan int
	disconnects chan int
	events      chan string
	unicast     Unicast
	nextResult  error
}

func (c *testContext) Command(command Command) error {
	c.commands <- command
	result := c.nextResult
	c.nextResult = nil
	return result
}

func (c *testContext) Connected(clientId int) {
	c.connects <- clientId
}

func (c *testContext) Disconnected(clientId int) {
	c.disconnects <- clientId
}

func (c *testContext) waitForEvent(t *testing.T, event string) {
	for event := range c.events {
		if event == event {
			return
		}
	}

	t.Fail()
}

func runTestServer(t *testing.T) (func(), string, *testContext) {
	ctx, cancel := context.WithCancel(context.Background())
	logger, _ := zap.NewDevelopment()
	server := NewServer(logger, prometheus.NewRegistry())
	testCtx := &testContext{
		make(chan Command, 100),
		make(chan int, 100),
		make(chan int, 100),
		make(chan string, 100),
		server,
		nil,
	}

	port, err := freeport.GetFreePort()

	require.NoError(t, err)

	address := fmt.Sprintf("localhost:%d", port)

	done := make(chan error, 1)

	go func() {
		done <- server.Run(ctx, address, testCtx)
	}()

	stop := func() {
		cancel()
		<-done
	}

	return stop, address, testCtx
}

func withServer(t *testing.T, action func(*testContext, string)) {
	stop, address, ctx := runTestServer(t)

	defer stop()

	action(ctx, address)
}

func withSession(t *testing.T, action func(*testContext, *Session)) {
	withServer(t, func(ctx *testContext, address string) {
		session, err := NewSession(address, ctx.events)

		defer session.Close()

		assert.Nil(t, err)

		action(ctx, session)
	})
}

func (s *Session) sendCommandAndExpectOK(t *testing.T, name string, args []string) {
	result, err := s.SendCommand(name, args)
	assert.True(t, result)
	assert.Nil(t, err)
}

func (s *Session) join(t *testing.T, name string) {
	s.sendCommandAndExpectOK(t, "join", []string{name})
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

		context.unicast.SendTo(clientId, "hello")

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

func TestNotifiesClientsOnShutdown(t *testing.T) {
	var (
		session *Session
		testCtx *testContext
	)

	withServer(t, func(ctx *testContext, address string) {
		var err error

		session, err = NewSession(address, ctx.events)

		assert.Nil(t, err)

		testCtx = ctx
	})

	defer session.Close()

	testCtx.waitForEvent(t, MessageServerIsShuttingDown)
}
