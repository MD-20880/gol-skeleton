package BrokerService

type eventCommand uint8

const (
	GetMap eventCommand = iota
	HandlerStop
)

type EventRequest interface {
	Command() eventCommand
}

//EventRequest
type GetMapEvent struct {
	Cmd      eventCommand
	SendBack chan CurrentWorld
}

type HandlerStopEvent struct {
	Cmd eventCommand
}

//EventResponse
type CurrentWorld struct {
	World [][]byte
	Turn  int
}

func (e GetMapEvent) Command() (c eventCommand) {
	return e.Cmd
}

func (e HandlerStopEvent) Command() (c eventCommand) {
	return e.Cmd
}
