package gol

import (
	"fmt"
	"strconv"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.

	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	//Pass File name to IO part
	file := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- file

	//Receive image from IO Part
	for i := range world {
		for j := range world[i] {
			world[i][j] = <-c.ioInput
		}
	}

	turn := 0

	// TODO: Execute all turns of the Game of Life.
	chans := make([]chan [][]byte, p.Threads)
	for i := range chans {
		chans[i] = make(chan [][]byte)
	}
	//Run GOL implementation for TURN times.
	for turn = p.Turns; turn > 0; turn-- {
		var newWorld [][]byte
		for i := 0; i < p.Threads-1; i++ {
			go startWorker(p, world, i*p.ImageHeight/p.Threads, 0, (i+1)*p.ImageHeight/p.Threads, p.ImageWidth, chans[i])
		}
		go startWorker(p, world, (p.Threads-1)*p.ImageHeight/p.Threads, 0, p.ImageHeight, p.ImageWidth, chans[p.Threads-1])

		for i := range chans {
			tempStore := <-chans[i]
			newWorld = append(newWorld, tempStore...)
		}
		world = newWorld
	}

	// Get all Alive cells.
	aliveCells := calculateAliveCells(p, world)
	fmt.Println(aliveCells)

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          aliveCells,
	}

	// Make sure that the Io has finished any output before exiting.
	fmt.Printf("checking idle\n")
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	fmt.Printf("closing\n")
	close(c.events)
}
