package transport

type Command struct {
	ClientId int
	Name     string
	Args     []string
}

type Unicast interface {
	SendTo(int, string)
}

type CommandHandler interface {
	Command(Command) error

	Connected(clientId int)
	Disconnected(clientId int)
}
