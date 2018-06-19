package main

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

type user struct {
	id   int
	name string
}

type Chat struct {
	users   map[int]user
	unicast unicast
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

func NewChat(unicast unicast) *Chat {
	return &Chat{
		make(map[int]user, 128),
		unicast,
	}
}

func (chat *Chat) connected(clientId int) {
	chat.unicast.sendTo(clientId, "Welcome!")
}

func (chat *Chat) disconnected(clientId int) {
	chat.leave(clientId)
}

func (chat *Chat) command(command command) error {
	switch command.name {
	case commandJoin:
		return chat.join(command.clientId, command.args)
	case commandSay:
		return chat.say(command.clientId, command.args)
	case commandRename:
		return chat.rename(command.clientId, command.args)
	default:
		return ErrUnknownCommand
	}
}

func (chat *Chat) sendToAll(message string) {
	for userId := range chat.users {
		chat.unicast.sendTo(userId, message)
	}
}

func (chat *Chat) rename(userId int, args []string) error {
	if len(args) != 1 {
		return ErrNameRequired
	}

	current, exists := chat.users[userId]

	if !exists {
		return ErrNotJoined
	}

	user := user{userId, args[0]}

	for _, existing := range chat.users {
		if existing.id != user.id && existing.name == user.name {
			return ErrNameNotUnique
		}
	}

	chat.users[user.id] = user
	chat.sendToAll(fmt.Sprintf("User %s changed his name to %s", current.name, user.name))

	return nil
}

func (chat *Chat) join(userId int, args []string) error {
	if len(args) != 1 {
		return ErrNameNotUnique
	}

	_, exists := chat.users[userId]

	if exists {
		return ErrAlreadyJoined
	}

	user := user{userId, args[0]}

	for _, existing := range chat.users {
		if existing.name == user.name {
			return ErrNameNotUnique
		}
	}

	chat.users[user.id] = user
	chat.sendToAll(fmt.Sprintf("User %s joined", user.name))

	return nil
}

func (chat *Chat) leave(userId int) {
	user, found := chat.users[userId]

	if !found {
		return
	}

	delete(chat.users, user.id)
	chat.sendToAll(fmt.Sprintf("User %s left", user.name))
}

func (chat *Chat) say(userId int, args []string) error {
	if len(args) < 1 {
		return ErrMessageRequired
	}

	user, found := chat.users[userId]

	if !found {
		return ErrNotJoined
	}

	text := strings.Join(args, " ")

	chat.sendToAll(fmt.Sprintf("%s: %s", user.name, text))

	return nil
}
