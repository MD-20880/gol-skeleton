package gol

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

var globalWorld [][]byte
var globalP Params
var turns int
var mutex = &sync.Mutex{}
var globalClient []*rpc.Client
var distributeChannels distributorChannels
var clientBroker *rpc.Client

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

func handleError(err error) {
	if err != nil {
		fmt.Println("something went wrong!")
	}
}

func countCell(world [][]byte) int {
	count := 0
	for i := 0; i < globalP.ImageHeight; i++ {
		for j := 0; j < globalP.ImageWidth; j++ {
			if world[i][j] == 255 {
				count += 1
			}
		}
	}
	return count
}

func CalculateAliveCells(world [][]byte) []util.Cell {
	container := make([]util.Cell, 0)
	//count := 0
	for i := 0; i < globalP.ImageWidth; i++ {
		for j := 0; j < globalP.ImageHeight; j++ {
			if world[i][j] == 255 {
				container = append(container, util.Cell{X: j, Y: i}) // had no key here before
				//container[count] = util.Cell{j, i}
				//count++
			}
		}
	}
	return container
}

// keyPresses, for p and k, a rpc call to one of broker's method to make it work distributedly
func keyPressesAction() {
	for {
		switch <-distributeChannels.keyPresses {
		case 's':
			outputPgm()
		case 'q':
			outputPgm()
			os.Exit(1)
		case 'p':
			response := new(stubs.StatusReport)
			clientBroker.Call(stubs.Pause, new(stubs.StatusReport), response)
			fmt.Println(response.Number)
			if <-distributeChannels.keyPresses == 'p' {
				clientBroker.Call(stubs.Pause, new(stubs.StatusReport), new(stubs.StatusReport))
				fmt.Println("continuing")
			}
		case 'k':
			clientBroker.Go(stubs.Kill, new(stubs.StatusReport), new(stubs.StatusReport), nil)
			outputPgm()
			os.Exit(1)
		}
	}
}

// send aliveCellCount every 2 seconds
func tickers(event chan<- Event, done chan bool) {
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			response := new(stubs.Response)
			clientBroker.Call(stubs.GetWorld, new(stubs.StatusReport), response)
			event <- AliveCellsCount{CompletedTurns: response.Turns, CellsCount: countCell(response.World)}
		case <-done:
			return
		}
	}
}

// output the pgm file with the current state
func outputPgm() {
	distributeChannels.ioCommand <- ioOutput
	response := new(stubs.Response)
	clientBroker.Call(stubs.GetWorld, new(stubs.StatusReport), response)
	outputString := strconv.Itoa(globalP.ImageHeight) + "x" + strconv.Itoa(globalP.ImageWidth) + "x" + strconv.Itoa(response.Turns)
	distributeChannels.ioFilename <- outputString
	world := response.World
	for i := 0; i < globalP.ImageHeight; i++ {
		for j := 0; j < globalP.ImageWidth; j++ {
			distributeChannels.ioOutput <- world[i][j]
		}
	}
	distributeChannels.events <- ImageOutputComplete{CompletedTurns: response.Turns, Filename: outputString}
}

// was a useful function in previous versions
func readFile() []string {
	f, err := os.Open("serverList")
	handleError(err)
	slice := make([]string, 2)
	scanner := bufio.NewScanner(f)
	i := 0
	for scanner.Scan() {
		slice[i] = scanner.Text()
	}
	return slice
}

// distributor publish work and make connections and calls to the broker
// as well as interacting with other goroutines.
func distributor(p Params, c distributorChannels) {

	// preparation work
	done := make(chan bool)
	go tickers(c.events, done)
	go keyPressesAction()
	globalP = p
	globalWorld = createWorld()
	turns = 0 // default is 0 I think
	distributeChannels = c

	c.ioCommand <- ioInput
	string := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioFilename <- string
	getInput(c)

	// making connection to the broker
	dAddr := "8040"
	//tempClient, err := rpc.Dial("tcp", "127.0.0.1:8030")
	tempClient, err := rpc.Dial("tcp", "184.73.70.65:8030")
	clientBroker = tempClient
	defer tempClient.Close()
	handleError(err)
	rpc.Register(&Distributor{})
	listener, err2 := net.Listen("tcp", ":"+dAddr)
	defer listener.Close()
	handleError(err2)
	go rpc.Accept(listener)

	// making rpc call to the broker to publish all its work
	request := stubs.Request{Turns: p.Turns, ImageWidth: p.ImageWidth, ImageHeight: p.ImageHeight, World: globalWorld, Address: "127.0.0.1:" + dAddr}
	response := new(stubs.Response)
	clientBroker.Call(stubs.Publish, request, response)

	// standardised stuffs
	done <- true
	if response.World != nil {
		event := FinalTurnComplete{CompletedTurns: globalP.Turns, Alive: CalculateAliveCells(response.World)}
		c.events <- event
	}
	outputPgm()
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

type Distributor struct{}

// CheckConnection two-way rpc calls, to check if the connection is still on
func (d *Distributor) CheckConnection(req stubs.StatusReport, res *stubs.StatusReport) (err error) {
	res.Message = "zhu"
	return
}
