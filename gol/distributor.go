package gol

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
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
var distributeChannels distributorChannels

func createWorld() [][]byte {
	world := make([][]byte, globalP.ImageHeight)
	for i := range world {
		world[i] = make([]byte, globalP.ImageWidth)
	}
	return world
}

//TODO current state so how about the one is still processing need of mutex lock?
// buggy code need to get to a lab machine to test it out
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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	// step 3 ticker part
	done := make(chan bool)
	go tickers(c.events, done)
	globalP = p
	globalWorld = createWorld()
	turns = 0
	distributeChannels = c

	// step 1 command and filename
	c.ioCommand <- ioInput
	string := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioFilename <- string

	for i := range globalWorld {
		for j := range globalWorld[i] {
			globalWorld[i][j] = <-c.ioInput
			if globalWorld[i][j] == 255 {
				c.events <- CellFlipped{CompletedTurns: turns, Cell: util.Cell{X: j, Y: i}}
			}
		}
	}

	go keyPressesAction()

	channels := createChannels(p)
	unitLength := p.ImageHeight / p.Threads
	for i := 0; i < p.Turns; i++ {
		j := 0
		for j < p.Threads-1 {
			go calculateNextState(p, globalWorld, j*unitLength, (j+1)*unitLength, 0, p.ImageWidth, channels[j])
			j++ // wonder if j++ works
		}
		go calculateNextState(p, globalWorld, j*unitLength, p.ImageHeight, 0, p.ImageWidth, channels[p.Threads-1])
		mutex.Lock()
		globalWorld = combine(channels, p)
		mutex.Unlock()
		turns++
		//for i := range globalWorld {
		//	for j := range globalWorld[i] {
		//
		//		if globalWorld[i][j] != nextWorld[i][j] {
		//			c.events <- CellFlipped{CompletedTurns: turns, Cell: util.Cell{X: j, Y: i}}
		//		}
		//	}
		//}
		//globalWorld = nextWorld
		c.events <- TurnComplete{CompletedTurns: turns}
	}
	done <- true
	//fmt.Println(globalWorld)

	event := FinalTurnComplete{CompletedTurns: p.Turns, Alive: calculateAliveCells(p, globalWorld)}
	c.events <- event

	outputPgm()
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func output() {

}

func tickers(event chan<- Event, done chan bool) {
	//event <- AliveCellsCount{CompletedTurns: turns, CellsCount: countCell()}
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			event <- AliveCellsCount{CompletedTurns: turns, CellsCount: countCell()}
			// find the problem, so when calculating it counts all the points in the world
			// but maybe when it was counting, the world updated, now it counts the new world therefore the problem
		case <-done:
			return
		}
	}
}

// want to refactor later 1. for range 2. could use channel dunno the implications
func countCell() int {
	mutex.Lock()
	world := globalWorld
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

func createChannels(p Params) []chan [][]byte {
	channels := make([]chan [][]byte, p.Threads)
	for i := range channels {
		channels[i] = make(chan [][]byte)
	}
	return channels
}

func combine(channels []chan [][]byte, p Params) [][]byte {
	var ultimateSlice [][]uint8
	for i := 0; i < p.Threads; i++ {
		v := <-channels[i]
		ultimateSlice = append(ultimateSlice, v...)
	}
	return ultimateSlice
}

func worker(p Params, world [][]byte, startY, endY, startX, endX int) {

}
