package chat

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/shse/go-chat/transport"
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

func (u testUnicast) SendTo(clientId int, message string) {
	u.messages <- unicastMessage{clientId, message}
}

func (u testUnicast) waitForMessage(t *testing.T, clientId int, data string) {
	for message := range u.messages {
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

func (c *Chat) executeCommandAndExpectSuccess(t *testing.T, clientId int, name string, args []string) {
	assert.Nil(t, c.Command(transport.Command{clientId, name, args}))
}

func (c *Chat) executeCommandAndExpectError(t *testing.T, clientId int, name string, args []string) (err error) {
	err = c.Command(transport.Command{ClientId: clientId, Name: name, Args: args})
	assert.Error(t, err)
	return
}

func TestSendsGreetingWhenConnected(t *testing.T) {
	chat, u := testChat()

	chat.Connected(1)

	assert.Equal(t, unicastMessage{1, "Welcome!"}, <-u.messages)
}

func TestNotifiesEveryoneAfterUserJoined(t *testing.T) {
	chat, u := testChat()

	chat.Connected(1)
	err := chat.Command(transport.Command{ClientId: 1, Name: "join", Args: []string{"john"}})

	assert.Nil(t, err)

	u.waitForMessage(t, 1, "User john joined")
}

func TestNotifiesEveryoneAfterUserLeft(t *testing.T) {
	chat, u := testChat()

	chat.Connected(1)
	chat.Connected(2)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectSuccess(t, 2, "join", []string{"alex"})

	chat.Disconnected(1)

	u.waitForMessage(t, 2, "User john left")
}

func TestDoesNotAllowToJoinIfNameAlreadyExists(t *testing.T) {
	chat, _ := testChat()

	chat.Connected(1)
	chat.Connected(2)

	var err error

	err = chat.Command(transport.Command{ClientId: 1, Name: "join", Args: []string{"john"}})

	assert.Nil(t, err)

	err = chat.Command(transport.Command{ClientId: 2, Name: "join", Args: []string{"john"}})

	assert.NotNil(t, err)
}

func TestDoesNotAllowToRenameIfNameAlreadyExists(t *testing.T) {
	chat, _ := testChat()

	chat.Connected(1)
	chat.Connected(2)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectSuccess(t, 2, "join", []string{"alex"})

	chat.executeCommandAndExpectError(t, 1, "rename", []string{"alex"})
}

func TestNotifiesEveryoneAfterUserChangedHisName(t *testing.T) {
	chat, u := testChat()

	chat.Connected(1)
	chat.Connected(2)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectSuccess(t, 2, "join", []string{"alex"})

	chat.executeCommandAndExpectSuccess(t, 1, "rename", []string{"tom"})

	u.waitForMessage(t, 2, "User john changed his name to tom")
}

func TestDeliversUserMessages(t *testing.T) {
	chat, u := testChat()

	chat.Connected(1)
	chat.Connected(2)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectSuccess(t, 2, "join", []string{"alex"})

	chat.executeCommandAndExpectSuccess(t, 1, "say", []string{"hello"})

	u.waitForMessage(t, 2, "john: hello")
}

func TestReturnsErrorWhenJoinTwice(t *testing.T) {
	chat, _ := testChat()

	chat.Connected(1)

	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})
	chat.executeCommandAndExpectError(t, 1, "join", []string{"john"})
}

func TestReturnsErrorWhenJoinWithoutName(t *testing.T) {
	chat, _ := testChat()

	chat.Connected(1)

	chat.executeCommandAndExpectError(t, 1, "join", []string{})
}

func TestReturnsErrorWhenCallSayWithoutText(t *testing.T) {
	chat, _ := testChat()

	chat.Connected(1)
	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})

	chat.executeCommandAndExpectError(t, 1, "say", []string{})
}

func TestReturnsErrorWhenRenameWithoutName(t *testing.T) {
	chat, _ := testChat()

	chat.Connected(1)
	chat.executeCommandAndExpectSuccess(t, 1, "join", []string{"john"})

	chat.executeCommandAndExpectError(t, 1, "rename", []string{})
}
