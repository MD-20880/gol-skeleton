package gol

import (
	"fmt"
	"net/rpc"
	"strconv"
	"uk.ac.bris.cs/gameoflife/stubs"
)

//TODO most important of all, add error handling to it
// Would had save me tons of time
type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

var globalWorld [][]byte
var globalP Params
var turns int

func createWorld() [][]byte {
	world := make([][]byte, globalP.ImageHeight)
	for i := range world {
		world[i] = make([]byte, globalP.ImageWidth)
	}
	return world
}

func getInput(c distributorChannels) {
	for i := range globalWorld {
		for j := range globalWorld[i] {
			globalWorld[i][j] = <-c.ioInput
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	globalP = p
	globalWorld = createWorld()
	turn := 0

	c.ioCommand <- ioInput
	string := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioFilename <- string

	getInput(c)

	// fixing rn
	server := "127.0.0.1:8030"
	//flag.Parse()
	fmt.Println("Server: ", server)
	client, _ := rpc.Dial("tcp", server)
	defer client.Close()
	response := makeCall(*client)

	event := FinalTurnComplete{CompletedTurns: globalP.Turns, Alive: response.AliveCells}
	c.events <- event

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

//TODO not sure if the global v would cause any troubles
func makeCall(client rpc.Client) stubs.Response {
	request := stubs.Request{Turns: globalP.Turns, ImageWidth: globalP.ImageWidth, ImageHeight: globalP.ImageHeight, World: globalWorld}
	response := new(stubs.Response)
	client.Call(stubs.GolHandler, request, response)
	return *response
}
