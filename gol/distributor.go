package gol

import (
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

	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}
	turn := 0

	c.ioCommand <- ioInput
	string := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioFilename <- string

	for i := range world {
		for j := range world[i] {
			world[i][j] = <-c.ioInput
		}
	}

	channels := createChannels(p)
	unitLength := p.ImageHeight / p.Threads
	for i := 0; i < p.Turns; i++ {
		j := 0
		for j < p.Threads-1 {
			go calculateNextState(p, world, j*unitLength, (j+1)*unitLength, 0, p.ImageWidth, channels[j])
			j++ // wonder if j++ works
		}
		go calculateNextState(p, world, j*unitLength, p.ImageHeight, 0, p.ImageWidth, channels[p.Threads-1])
		world = combine(channels, p)
	}
	//fmt.Println(world)
	event := FinalTurnComplete{CompletedTurns: p.Turns, Alive: calculateAliveCells(p, world)}
	c.events <- event

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
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
