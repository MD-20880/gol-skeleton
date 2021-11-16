package gol

import (
	"bufio"
	"fmt"
	"github.com/ChrisGora/semaphore"
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

func getServerList() {
	connMap = map[string]*rpc.Client{}
	serverList = make([]string, 0)
	scanner := readfile("gol/serverList")
	for scanner.Scan() {
		serverList = append(serverList, scanner.Text())
	}
	for _, server := range serverList {
		connMap[server], _ = rpc.Dial("tcp", server)
	}
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

func updateTurn(chans []chan [][]byte) [][]byte {
	var updatedWorld [][]byte
	for i := 0; i < p.Threads-1; i++ {
		go startWorker(i*p.ImageHeight/p.Threads, 0, (i+1)*p.ImageHeight/p.Threads, p.ImageWidth, chans[i], serverList[i%(len(serverList))])
	}
	go startWorker((p.Threads-1)*p.ImageHeight/p.Threads, 0, p.ImageHeight, p.ImageWidth, chans[p.Threads-1], serverList[0])

	for i := range chans {
		tempStore := <-chans[i]
		updatedWorld = append(updatedWorld, tempStore...)
	}

	return updatedWorld

}

func startWorker(startX int, startY int, endX int, endY int, resultchan chan [][]byte, server string) {
	req := stubs.Request{
		Turns:        turn,
		ImageWidth:   p.ImageWidth,
		ImageHeight:  p.ImageHeight,
		StartX:       startX,
		StartY:       startY,
		EndX:         endX,
		EndY:         endY,
		CalculateMap: world,
	}
	rsp := new(stubs.Response)
	connMap[server].Call(stubs.Calculate, req, rsp)
	resultchan <- rsp.Result
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
func checkFlipCells(oldWorld *[][]byte, newWorld *[][]byte, p Params) []util.Cell {
	oldCells := CalculateAliveCells(*oldWorld)
	newCells := CalculateAliveCells(*newWorld)
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
			os.Exit(1)
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
	close(c.events)
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(params Params, channels distributorChannels, avail *channelAvailibility, keyPressed <-chan rune) {
	p = params
	c = channels
	a = *avail
	semaPhore = semaphore.Init(1, 1)
	getServerList()

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
	for i := range world {
		for j := range world[i] {
			if world[i][j] == 255 {
				c.events <- CellFlipped{turn, util.Cell{i, j}}
			}
		}
	}

	conn, _ := rpc.Dial("tcp", "127.0.0.1:8030")
	defer conn.Close()
	//c.events <- TurnComplete{CompletedTurns: turn}

	//test

	//req := stubs.PublishTask{
	//	GolMap:      world,
	//	Turns:       p.Turns,
	//	ImageWidth:  p.ImageWidth,
	//	ImageHeight: p.ImageHeight,
	//}
	//
	//res := new(stubs.GolResultReport)
	//conn.Call(stubs.DistributorPublish, req, res)
	//newWorld = res.ResultMap

	//Run GOL implementation for TURN times.
	for i := 1; i <= p.Turns; i++ {
		semaPhore.Wait()

		//newWorld = updateTurn(chans)
		req := stubs.PublishTask{
			GolMap:      world,
			Turns:       1,
			ImageWidth:  p.ImageWidth,
			ImageHeight: p.ImageHeight,
		}

		res := new(stubs.GolResultReport)
		conn.Call(stubs.DistributorPublish, req, res)
		newWorld = res.ResultMap
		//stupid function
		//flipCells := checkFlipCells(&world,&newWorld,p)
		//smart one
		flipCells := newCheckFlipCells()
		for j := range flipCells {
			c.events <- CellFlipped{turn, flipCells[j]}
		}
		c.events <- TurnComplete{CompletedTurns: turn}
		//cell Flipped event
		mutex.Lock()
		world = newWorld
		turn = i
		mutex.Unlock()
		semaPhore.Post()
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	quit()
}
