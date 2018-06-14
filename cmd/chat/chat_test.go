package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type unicastMessage struct {
	clientId int
	message  string
}

type testUnicast struct {
	messages chan unicastMessage
}

func newTestUnicast() testUnicast {
	return testUnicast{
		make(chan unicastMessage, 100),
	}
}

func (unicast testUnicast) sendTo(clientId int, message string) {
	unicast.messages <- unicastMessage{clientId, message}
}

func (unicast testUnicast) waitForMessage(t *testing.T, clientId int, data string) {
	for message := range unicast.messages {
		fmt.Printf("Received message for %d: %s\n", message.clientId, message.message)

		if message.clientId == clientId && message.message == data {
			return
		}
	}

	t.Fail()
}

func testChat() (*Chat, testUnicast) {
	u := newTestUnicast()

	chat := NewChat(u)

	return chat, u
}

func (chat *Chat) executeCommandAndExpectSuccess(t *testing.T, clientId int, name string, args []string) {
	assert.Nil(t, chat.command(command{clientId, name, args}))
}

func (chat *Chat) executeCommandAndExpectError(t *testing.T, clientId int, name string, args []string) (err error) {
	err = chat.command(command{clientId, name, args})
	assert.NotNil(t, err)
	return
}

func TestSendsGreetingWhenConnected(t *testing.T) {
	chat, u := testChat()

	chat.connected(1)

	assert.Equal(t, unicastMessage{1, "Welcome!"}, <-u.messages)
}

func TestNotifiesEveryoneAfterUserJoined(t *testing.T) {
	chat, u := testChat()

	chat.connected(1)
	err := chat.command(command{1, "join", []string{"john"}})

	assert.Nil(t, err)

	u.waitForMessage(t, 1, "User john joined")
}

func TestNotifiesEveryoneAfterUserLeft(t *testing.T) {
	chat, u := testChat()

	chat.connected(1)
	chat.connected(2)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectSuccess(t, 2, "join", []string{"alex"})

	chat.disconnected(1)

	u.waitForMessage(t, 2, "User john left")
}

func TestDoesNotAllowToJoinIfNameAlreadyExists(t *testing.T) {
	chat, _ := testChat()

	chat.connected(1)
	chat.connected(2)

	var err error

	err = chat.command(command{1, "join", []string{"john"}})

	assert.Nil(t, err)

	err = chat.command(command{2, "join", []string{"john"}})

	assert.NotNil(t, err)
}

func TestDoesNotAllowToRenameIfNameAlreadyExists(t *testing.T) {
	chat, _ := testChat()

	chat.connected(1)
	chat.connected(2)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectSuccess(t, 2, "join", []string{"alex"})

	chat.executeCommandAndExpectError(t, 1, "rename", []string{"alex"})
}

func TestNotifiesEveryoneAfterUserChangedHisName(t *testing.T) {
	chat, u := testChat()

	chat.connected(1)
	chat.connected(2)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectSuccess(t, 2, "join", []string{"alex"})

	chat.executeCommandAndExpectSuccess(t, 1, "rename", []string{"tom"})

	u.waitForMessage(t, 2, "User john changed his name to tom")
}

func TestDeliversUserMessages(t *testing.T) {
	chat, u := testChat()

	chat.connected(1)
	chat.connected(2)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectSuccess(t, 2, "join", []string{"alex"})

	chat.executeCommandAndExpectSuccess(t, 1, "say", []string{"hello"})

	u.waitForMessage(t, 2, "john: hello")
}
