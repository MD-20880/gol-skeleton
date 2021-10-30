package gol

import (
	"fmt"
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
}

type channelAvailibility struct {
	events     bool
	ioCommand  bool
	ioIdle     bool
	ioFilename bool
	ioOutput   bool
	ioInput    bool
}

func inLoopEventHandler(c distributorChannels, p Params, turn *int, currentWorld *[][]byte, mutex *sync.Mutex, a *channelAvailibility) {
	var result []util.Cell
	world := *currentWorld
	//Count report Event
	go reportCount(c, p, turn, &result, mutex, a)
	for {
		mutex.Lock()
		result = calculateAliveCells(p, world)
		flipList := make([]util.Cell, 0)
		//check every block
		cellIndex := 0
		for i := range world {
			for j := range world[i] {
				if cellIndex >= len(result) {
					break
				}
				//if It was alive, then check whether it is alive
				if world[i][j] == 255 {
					//I assume everything will be in order here
					if result[cellIndex].X == i && result[cellIndex].Y == j {
						continue
					} else {
						flipList = append(flipList, util.Cell{
							X: i,
							Y: j,
						})
						cellIndex++
					}
				}
			}
		}
		for info := range flipList {
			c.events <- CellFlipped{
				CompletedTurns: *turn,
				Cell:           flipList[info],
			}
		}

		c.events <- TurnComplete{CompletedTurns: *turn}
		mutex.Unlock()

	}
}

func reportCount(c distributorChannels, p Params, turn *int, result *[]util.Cell, mutex *sync.Mutex, a *channelAvailibility) {
	for {
		time.Sleep(2 * time.Second)
		if a.events == true {
			mutex.Lock()
			c.events <- AliveCellsCount{
				CompletedTurns: p.Turns - *turn,
				CellsCount:     len(*result),
			}
			mutex.Unlock()
		} else {
			return
		}
	}

}

func storePgm(mutex sync.Mutex, world *[][]byte, c distributorChannels, p Params) {
	c.ioCommand <- ioOutput
	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.Turns)
	c.ioFilename <- filename
	mutex.Lock()
	currentWorld := *world
	for i := range currentWorld {
		for j := range currentWorld[i] {
			c.ioOutput <- currentWorld[i][j]
		}
	}
	mutex.Unlock()
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, a *channelAvailibility) {

	mutex := sync.Mutex{}

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

	//Task 3
	//go inLoopEventHandler(c, p, &turn, &world, &mutex,a)

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
		//cell Flipped event

		mutex.Lock()
		world = newWorld
		mutex.Unlock()
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	// Get all Alive cells.
	aliveCells := calculateAliveCells(p, world)
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          aliveCells,
	}
	fmt.Println("Start writing")
	storePgm(mutex, &world, c, p)
	// Make sure that the Io has finished any output before exiting.
	fmt.Printf("checking idle\n")
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	fmt.Printf("closing\n")
	a.events = false
	close(c.events)
}
