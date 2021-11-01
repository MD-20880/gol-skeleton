package gol

import (
	"fmt"
	"github.com/ChrisGora/semaphore"
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

var p Params
var world [][]byte
var mutex sync.Mutex
var a channelAvailibility
var c distributorChannels
var turn int
var semaPhore semaphore.Semaphore

func reportCount() {
	for {
		result := calculateAliveCells(p, world)
		time.Sleep(2 * time.Second)
		if a.events == true {
			semaPhore.Wait()
			c.events <- AliveCellsCount{
				CompletedTurns: p.Turns - turn,
				CellsCount:     len(result),
			}
			semaPhore.Post()
		} else {
			return
		}
	}

}

//This function Work just well
func storePgm() {
	c.ioCommand <- ioOutput
	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.Turns-turn)
	c.ioFilename <- filename
	mutex.Lock()
	currentWorld := world
	for i := range currentWorld {
		for j := range currentWorld[i] {
			c.ioOutput <- currentWorld[i][j]
		}
	}
	mutex.Unlock()
}

func updateTurn(chans []chan [][]byte) [][]byte {
	var newWorld [][]byte
	for i := 0; i < p.Threads-1; i++ {
		go startWorker(p, world, i*p.ImageHeight/p.Threads, 0, (i+1)*p.ImageHeight/p.Threads, p.ImageWidth, chans[i])
	}
	go startWorker(p, world, (p.Threads-1)*p.ImageHeight/p.Threads, 0, p.ImageHeight, p.ImageWidth, chans[p.Threads-1])

	for i := range chans {
		tempStore := <-chans[i]
		newWorld = append(newWorld, tempStore...)
	}

	return newWorld

}

func cellsGreaterThan(a util.Cell, b util.Cell) bool {
	if a.X > b.X {
		return true
	} else if a.Y > b.Y {
		return true
	} else {
		return false
	}
}
func cellEqual(a util.Cell, b util.Cell) bool {
	if a.Y == b.Y && a.X == b.X {
		return true
	}
	return false
}

//TODO : I just want to remind you that this function sucks.
func checkFlipCells(oldWorld *[][]byte, newWorld *[][]byte, p Params) []util.Cell {
	oldCells := calculateAliveCells(p, *oldWorld)
	newCells := calculateAliveCells(p, *newWorld)
	flipCells := make([]util.Cell, 0)
	i := 0
	j := 0
	for i < len(oldCells) && j < len(newCells) {
		if cellEqual(oldCells[i], newCells[j]) {
			i++
			j++
		} else if cellsGreaterThan(oldCells[i], newCells[j]) {
			flipCells = append(flipCells, newCells[j])
			j++
		} else {
			flipCells = append(flipCells, oldCells[i])
			i++
		}
	}
	if i < len(oldCells) {
		addCell := oldCells[i:len(oldCells)]
		flipCells = append(flipCells, addCell...)
	} else if j < len(newCells) {
		addCell := newCells[j:len(newCells)]
		flipCells = append(flipCells, addCell...)
	}
	return flipCells
}

func newCheckFlipCells(oldWorldP *[][]byte, newWorldP *[][]byte) []util.Cell {
	oldWorld := *oldWorldP
	newWorld := *newWorldP
	flipCells := make([]util.Cell, 0)
	for i := range oldWorld {
		for j := range oldWorld[i] {
			if oldWorld[i][j] != newWorld[i][j] {
				flipCells = append(flipCells, util.Cell{X: j, Y: i})
			}
		}
	}
	return flipCells
}

func checkKeyPressed(keyPressed <-chan rune) {
	for {
		i := <-keyPressed
		semaPhore.Wait()
		switch i {
		case 's':
			storePgm()
		case 'p':
			{
				key := <-keyPressed
				for key != 'p' {
					key = <-keyPressed
				}
				fmt.Printf("Continuing\n")
			}
		case 'q':
			quit()
		}
		semaPhore.Post()

	}
}

func quit() {
	aliveCells := calculateAliveCells(p, world)
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          aliveCells,
	}
	storePgm()
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	a.events = false
	close(c.events)
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(params Params, channels distributorChannels, avail *channelAvailibility, keyPressed <-chan rune) {
	p = params
	c = channels
	a = *avail
	semaPhore = semaphore.Init(1, 1)
	mutex = sync.Mutex{}

	// TODO: Create a 2D slice to store the world.
	world = make([][]byte, p.ImageHeight)
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

	turn = 0

	// TODO: Execute all turns of the Game of Life.
	chans := make([]chan [][]byte, p.Threads)
	for i := range chans {
		chans[i] = make(chan [][]byte)
	}

	//Task 3
	go reportCount()
	go checkKeyPressed(keyPressed)
	//Run GOL implementation for TURN times.
	for turn = p.Turns; turn > 0; turn-- {
		semaPhore.Wait()
		newWorld := updateTurn(chans)
		//stupid function
		//flipCells := checkFlipCells(&world,&newWorld,p)
		//smart one
		flipCells := newCheckFlipCells(&world, &newWorld)
		for i := range flipCells {
			c.events <- CellFlipped{turn, flipCells[i]}
		}
		c.events <- TurnComplete{CompletedTurns: turn}
		//cell Flipped event
		mutex.Lock()
		world = newWorld
		mutex.Unlock()
		semaPhore.Post()
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	quit()
}
