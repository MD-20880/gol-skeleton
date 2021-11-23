package gol

import (
	"bufio"
	"fmt"
	"github.com/ChrisGora/semaphore"
	"math/rand"
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
var newWorld [][]byte
var mutex = sync.Mutex{}
var a channelAvailibility
var c distributorChannels
var turn int
var semaPhore semaphore.Semaphore
var serverList []string
var connMap map[string]*rpc.Client

//Distributed Functions
func requestMap(id string, conn *rpc.Client) (world [][]byte, turn int) {
	req := stubs.RequestCurrentWorld{ID: id}
	res := new(stubs.RespondCurrentWorld)
	conn.Call("Broker.Getmap", req, res)
	world = res.World
	turn = res.Turn
	return
}

func dreportCount(id string, conn *rpc.Client) {
	for {
		time.Sleep(2 * time.Second)
		currentWorld, currentTurn := requestMap(id, conn)
		mutex.Lock()
		result := CalculateAliveCells(currentWorld)
		mutex.Unlock()
		if a.events == true {
			c.events <- AliveCellsCount{
				CompletedTurns: currentTurn,
				CellsCount:     len(result),
			}
			c.events <- TurnComplete{CompletedTurns: currentTurn}

		} else {
			return
		}
	}
}

//Parallel Functions
func reportCount() {
	for {
		time.Sleep(2 * time.Second)
		mutex.Lock()
		result := CalculateAliveCells(world)
		currentTurn := turn
		mutex.Unlock()
		if a.events == true {
			c.events <- AliveCellsCount{
				CompletedTurns: currentTurn,
				CellsCount:     len(result),
			}

		} else {
			return
		}
	}

}

func readfile(path string) bufio.Scanner {
	file, err := os.Open(path)
	//if err != nil{
	//	os.Exit(3)
	//}
	fmt.Println(err)
	scanner := bufio.NewScanner(file)
	return *scanner

}

//This function Work just well
func storePgm() {
	c.ioCommand <- ioOutput
	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(turn)
	c.ioFilename <- filename
	for i := range world {
		for j := range world[i] {
			c.ioOutput <- world[i][j]
		}
	}
}

func CalculateAliveCells(world [][]byte) []util.Cell {
	var cells = []util.Cell{}
	for j, _ := range world {
		for i, num := range world[j] {
			if num == 255 {
				cells = append(cells, util.Cell{i, j})
			}
		}
	}
	return cells
}

//TODO : I just want to remind you that this function sucks.

func newCheckFlipCells() []util.Cell {

	flipCells := make([]util.Cell, 0)
	for i := range world {
		for j := range world[i] {
			if world[i][j] != newWorld[i][j] {
				flipCells = append(flipCells, util.Cell{X: i, Y: j})
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
			//os.Exit(1)
		}
		semaPhore.Post()

	}
}

func quit() {
	aliveCells := CalculateAliveCells(world)
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          aliveCells,
	}
	storePgm()
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	for _, j := range connMap {
		j.Close()
	}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	a.events = false

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(params Params, channels distributorChannels, avail *channelAvailibility, keyPressed <-chan rune) {
	p = params
	c = channels
	a = *avail
	semaPhore = semaphore.Init(1, 1)
	rand.Seed(time.Now().UnixNano())
	id := strconv.Itoa(rand.Int())

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
	//SDLWorkFlow(params,channels,avail,keyPressed)
	DistributedWorkFlow(keyPressed, id)
}
