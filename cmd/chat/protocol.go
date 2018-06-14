package main

type command struct {
	clientId int
	name     string
	args     []string
}

type unicast interface {
	sendTo(int, string)
}

type commandHandler interface {
	command(command) error

	connected(clientId int)
	disconnected(clientId int)
}
