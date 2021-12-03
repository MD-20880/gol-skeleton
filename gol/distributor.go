package gol

import (
	"fmt"
	"github.com/ChrisGora/semaphore"
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
}

type channelAvailibility struct {
	events     bool
	ioCommand  bool
	ioIdle     bool
	ioFilename bool
	ioOutput   bool
	ioInput    bool
}

/* Struct Variables
Params
__________________________________
    p			Parameters passed by User
	world 		State of last calculated world { Most of the read operation happends here, protected by mutex lock
	newWorld	Current Calculating world { Most of the Write Operation happens here, if you want to read from it, please use mutex lock}
	mutex 		mutex lock that protect shared variables.
	a 			A struct that store the availability of channels { You can check availability of a channel before using it }
	c 			A struct that store the channels.
	turn 		Store the turn of last calculated world. { If you want to update this number, use mutex lock and update it right before / after updating world
	semaphore 	Semaphore that used to halting main loop. { To stop main loop and handling events }
*/

type Variables struct {
	p         Params
	world     [][]byte
	newWorld  [][]byte
	mutex     sync.Mutex
	a         channelAvailibility
	c         distributorChannels
	turn      int
	semaphore semaphore.Semaphore
}

/* Function "reportCount" send AliveCellsCount Event every two seconds.
params
_______________________________
	v			Variables { shared variables to every functions, if not mentioned, this is the default parameter description }

*/

func reportCount(v *Variables) {
	for {
		time.Sleep(2 * time.Second)
		v.mutex.Lock()
		result := CalculateAliveCells(v.p, v.world)
		currentTurn := v.turn
		v.mutex.Unlock()
		if v.a.events == true {
			v.c.events <- AliveCellsCount{
				CompletedTurns: currentTurn,
				CellsCount:     len(result),
			}

		} else {
			return
		}
	}

}

/*Function "storePgm" communicate to IO routine and save last calculated world.

 */
func storePgm(v *Variables) {
	v.c.ioCommand <- ioOutput
	filename := strconv.Itoa(v.p.ImageWidth) + "x" + strconv.Itoa(v.p.ImageHeight) + "x" + strconv.Itoa(v.turn)
	v.c.ioFilename <- filename
	for i := range v.world {
		for j := range v.world[i] {
			v.c.ioOutput <- v.world[i][j]
		}
	}
}

/*Function "updateTurn" update world by 1 round.
params
________________________________________
	chans			A list of channels receving result. { In order to reduce the time for making channels, Initialization of these channels is not in this function }

return
________________________________________
	A world updated from v.world

*/
func updateTurn(chans []chan [][]byte, v *Variables) [][]byte {
	var newWorld [][]byte
	for i := 0; i < v.p.Threads; i++ {
		temp := i * v.p.ImageHeight
		go StartWorker(v.p, v.world, temp/v.p.Threads, 0, (temp+v.p.ImageHeight)/v.p.Threads, v.p.ImageWidth, chans[i])
	}

	for i := range chans {
		tempStore := <-chans[i]
		newWorld = append(newWorld, tempStore...)
	}

	return newWorld

}

/*Function "checkFlipCells"
return
________________________________________
	A List of cells that need to be flipped

*/

func checkFlipCells(v *Variables) []util.Cell {

	flipCells := make([]util.Cell, 0)
	for i := range v.world {
		for j := range v.world[i] {
			if v.world[i][j] != v.newWorld[i][j] {
				flipCells = append(flipCells, util.Cell{X: i, Y: j})
			}
		}
	}
	return flipCells
}

/*Function "checkKeyPressed" Handle keyPress events send through keyPressed channel
params
______________________________________
	keyPressed			Channel receives keyPress event

*/

func checkKeyPressed(keyPressed <-chan rune, v *Variables) {
	for {
		i := <-keyPressed
		v.semaphore.Wait()
		switch i {
		case 's':
			storePgm(v)
		case 'p':
			{
				key := <-keyPressed
				for key != 'p' {
					key = <-keyPressed
				}
				fmt.Printf("Continuing\n")
			}
		case 'q':
			quit(v)
			os.Exit(1)
		}
		v.semaphore.Post()

	}
}

/*Function "quit" Quitting behaviors

 */

func quit(v *Variables) {
	aliveCells := CalculateAliveCells(v.p, v.world)
	v.c.events <- FinalTurnComplete{
		CompletedTurns: v.turn,
		Alive:          aliveCells,
	}
	storePgm(v)
	// Make sure that the Io has finished any output before exiting.
	v.c.ioCommand <- ioCheckIdle
	<-v.c.ioIdle

	v.c.events <- StateChange{v.turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	v.a.events = false
	close(v.c.events)
}

/*Function "initVars" Initialize variables in struct Variables before distributor start.

 */

func initVars(params Params, channels distributorChannels, avail *channelAvailibility) Variables {
	p := params
	c := channels
	a := *avail
	semaPhore := semaphore.Init(1, 1)

	//Create a 2D empty world
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	///Read world From File

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

	return Variables{
		p:         p,
		world:     world,
		newWorld:  nil,
		mutex:     sync.Mutex{},
		a:         a,
		c:         c,
		turn:      0,
		semaphore: semaPhore,
	}

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(params Params, channels distributorChannels, avail *channelAvailibility, keyPressed <-chan rune) {
	//Initialization Start
	v := initVars(params, channels, avail)

	chans := make([]chan [][]byte, v.p.Threads)
	for i := range chans {
		chans[i] = make(chan [][]byte)
	}

	for i := range v.world {
		for j := range v.world[i] {
			if v.world[i][j] == 255 {
				v.c.events <- CellFlipped{v.turn, util.Cell{i, j}}
			}
		}
	}

	//Initialization End

	//start Allive Cell Count Reporter
	go reportCount(&v)
	//start KeyPress Event Handler
	go checkKeyPressed(keyPressed, &v)

	//Run GOL implementation for TURN times.
	for i := 1; i <= v.p.Turns; i++ {
		//Semaphore for controlling workflow
		v.semaphore.Wait()
		v.newWorld = updateTurn(chans, &v)
		flipCells := checkFlipCells(&v)
		for j := range flipCells {
			v.c.events <- CellFlipped{v.turn, flipCells[j]}
		}
		v.mutex.Lock()
		v.c.events <- TurnComplete{CompletedTurns: v.turn}
		//Update World info and protect it by mutex lock
		v.world = v.newWorld
		v.turn = i
		v.mutex.Unlock()
		v.semaphore.Post()
	}

	//quit after task
	quit(&v)
}
