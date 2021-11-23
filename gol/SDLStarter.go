package gol

import (
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func SDLWorkFlow(keyPressed <-chan rune, id string) {
	conn, _ := rpc.Dial("tcp", "127.0.0.1:8030")
	defer conn.Close()

	chans := make([]chan [][]byte, p.Threads)
	for i := range chans {
		chans[i] = make(chan [][]byte)
	}

	go reportCount()
	go checkKeyPressed(keyPressed)

	for i := range world {
		for j := range world[i] {
			if world[i][j] == 255 {
				c.events <- CellFlipped{turn, util.Cell{i, j}}
			}
		}
	}

	c.events <- TurnComplete{CompletedTurns: turn}

	//Run GOL implementation for TURN times.
	for i := 1; i <= p.Turns; i++ {
		semaPhore.Wait()

		//newWorld = updateTurn(chans)
		req := stubs.PublishTask{
			ID:          id,
			GolMap:      world,
			Turns:       1,
			ImageWidth:  p.ImageWidth,
			ImageHeight: p.ImageHeight,
		}

		res := new(stubs.GolResultReport)
		conn.Call(stubs.DistributorPublish, req, res)
		newWorld = res.ResultMap

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

	quit()
	close(c.events)
}
