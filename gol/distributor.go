package gol

import (
	"bufio"
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
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
		fmt.Println("shabi ba ni")
	}
}

func keyPressesAction() {
	for {
		switch <-distributeChannels.keyPresses {
		case 's':
			outputPgm()
		case 'q':
			outputPgm()
			os.Exit(1) // not sure about this part also do I need to report event or not
		case 'p':
			fmt.Println(turns)
			mutex.Lock()
			if <-distributeChannels.keyPresses == 'p' {
				fmt.Println("continuing")
				mutex.Unlock()
			}

		}
	}
}

func outputPgm() {
	distributeChannels.ioCommand <- ioOutput
	outputString := strconv.Itoa(globalP.ImageHeight) + "x" + strconv.Itoa(globalP.ImageWidth) + "x" + strconv.Itoa(globalP.Turns)
	distributeChannels.ioFilename <- outputString
	world := globalWorld
	for i := 0; i < globalP.ImageHeight; i++ {
		for j := 0; j < globalP.ImageWidth; j++ {
			distributeChannels.ioOutput <- world[i][j]
		}
	}
	distributeChannels.events <- ImageOutputComplete{CompletedTurns: turns, Filename: outputString}
}

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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	done := make(chan bool)
	go tickers(c.events, done)
	globalP = p
	globalWorld = createWorld()
	turns = 0 // default is 0 I think
	distributeChannels = c

	c.ioCommand <- ioInput
	string := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioFilename <- string

	getInput(c)

	/* don't need this anymore
	ipAddr := readFile()
	length := len(ipAddr)
	globalClient = make([]*rpc.Client, length)
	for i := 0; i < length; i++ {
		client, err := rpc.Dial("tcp", ipAddr[i])
		handleError(err)
		globalClient[i] = client
		defer globalClient[i].Close() // tf do i deal with this
	} */

	// ganjuezhegeyemeishayong
	//rpcChannel := make(chan stubs.Response)
	go keyPressesAction() // dunno if its okay to put it here

	shabi, err := rpc.Dial("tcp", "127.0.0.1:8030")
	clientBroker = shabi
	handleError(err)
	request := stubs.Request{Turns: p.Turns, ImageWidth: p.ImageWidth, ImageHeight: p.ImageHeight, World: globalWorld, Address: ""}
	response := new(stubs.Response)
	clientBroker.Call(stubs.Publish, request, response)
	// clientBroker.Call(stubs.Distribute, new(stubs.StatusReport), new(stubs.StatusReport))
	// that part feels more like a temporary measure
	//for i := 0; i < p.Turns; i++ {
	//	for j := 0; j < length; j++ {
	//		go makeCall(*globalClient[j], rpcChannel, (p.ImageHeight / length) * (j + 1))
	//	}
	//	response := <-rpcChannel
	//	globalWorld = response.World
	//	turns++
	//}

	done <- true
	//here as well, global v might be problematic
	event := FinalTurnComplete{CompletedTurns: globalP.Turns, Alive: CalculateAliveCells(response.World)}
	c.events <- event
	globalWorld = response.World
	outputPgm()

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

//TODO not sure if the global v would cause any troubles
func makeCall(client rpc.Client, channel chan stubs.Response, length int) {
	request := stubs.Request{Turns: globalP.Turns, ImageWidth: globalP.ImageWidth, ImageHeight: globalP.ImageHeight, World: globalWorld}
	response := new(stubs.Response)
	client.Call(stubs.GolHandler, request, response)
	channel <- *response
}

func tickers(event chan<- Event, done chan bool) {
	//event <- AliveCellsCount{CompletedTurns: turns, CellsCount: countCell()}
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			response := new(stubs.Response)
			clientBroker.Call(stubs.GetWorld, new(stubs.StatusReport), response)
			// globalWorld = response.World
			event <- AliveCellsCount{CompletedTurns: response.Turns, CellsCount: countCell(response.World)} // turns haven't been done
			// find the problem, so when calculating it counts all the points in the world
			// but maybe when it was counting, the world updated, now it counts the new world therefore the problem
		case <-done:
			return
		}
	}
}

// want to refactor later 1. for range 2. could use channel dunno the implications
func countCell(world [][]byte) int {

	mutex.Lock()
	//world := globalWorld
	mutex.Unlock()
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
