package chat

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"github.com/shse/go-chat/transport"
)

type user struct {
	id   int
	name string
}

type Chat struct {
	users   map[int]user
	unicast transport.Unicast
}

const (
	commandJoin   = "join"
	commandSay    = "say"
	commandRename = "rename"
)

var (
	ErrNameNotUnique   = errors.New("name is not unique")
	ErrNameRequired    = errors.New("name required")
	ErrMessageRequired = errors.New("message required")
	ErrNotJoined       = errors.New("not joined")
	ErrUnknownCommand  = errors.New("unknown command")
	ErrAlreadyJoined   = errors.New("already joined")
)

func NewChat(unicast transport.Unicast) *Chat {
	return &Chat{
		make(map[int]user, 128),
		unicast,
	}
}

func (c *Chat) Connected(clientId int) {
	c.unicast.SendTo(clientId, "Welcome!")
}

func (c *Chat) Disconnected(clientId int) {
	c.leave(clientId)
}

func (c *Chat) Command(command transport.Command) error {
	switch command.Name {
	case commandJoin:
		return c.join(command.ClientId, command.Args)
	case commandSay:
		return c.say(command.ClientId, command.Args)
	case commandRename:
		return c.rename(command.ClientId, command.Args)
	default:
		return ErrUnknownCommand
	}
}

func (c *Chat) sendToAll(message string) {
	for userId := range c.users {
		c.unicast.SendTo(userId, message)
	}
}

func (c *Chat) rename(userId int, args []string) error {
	if len(args) != 1 {
		return ErrNameRequired
	}

	current, exists := c.users[userId]

	if !exists {
		return ErrNotJoined
	}

	user := user{userId, args[0]}

	for _, existing := range c.users {
		if existing.id != user.id && existing.name == user.name {
			return ErrNameNotUnique
		}
	}

	c.users[user.id] = user
	c.sendToAll(fmt.Sprintf("User %s changed his name to %s", current.name, user.name))

	return nil
}

func (c *Chat) join(userId int, args []string) error {
	if len(args) != 1 {
		return ErrNameRequired
	}

	_, exists := c.users[userId]

	if exists {
		return ErrAlreadyJoined
	}

	user := user{userId, args[0]}

	for _, existing := range c.users {
		if existing.name == user.name {
			return ErrNameNotUnique
		}
	}

	c.users[user.id] = user
	c.sendToAll(fmt.Sprintf("User %s joined", user.name))

	return nil
}

func (c *Chat) leave(userId int) {
	user, found := c.users[userId]

	if !found {
		return
	}

	delete(c.users, user.id)
	c.sendToAll(fmt.Sprintf("User %s left", user.name))
}

func (c *Chat) say(userId int, args []string) error {
	if len(args) < 1 {
		return ErrMessageRequired
	}

	user, found := c.users[userId]

	if !found {
		return ErrNotJoined
	}

	text := strings.Join(args, " ")

	c.sendToAll(fmt.Sprintf("%s: %s", user.name, text))

	return nil
}
