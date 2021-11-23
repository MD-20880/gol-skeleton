package gol

import (
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

func requestMap(id string, conn *rpc.Client) (world [][]byte, turn int) {
	req := stubs.RequestCurrentWorld{ID: id}
	res := new(stubs.RespondCurrentWorld)
	conn.Call("Broker.Getmap", req, res)
	world = res.World
	turn = res.Turn
	return
}

func dstorePgm(id string, conn *rpc.Client) {
	requestMap(id, conn)
	c.ioCommand <- ioOutput
	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(turn)
	c.ioFilename <- filename
	for i := range world {
		for j := range world[i] {
			c.ioOutput <- world[i][j]
		}
	}
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

func dcheckKeyPressed(keyPressed <-chan rune, id string) {
	for {
		i := <-keyPressed
		semaPhore.Wait()
		switch i {
		case 'k':
			{
				req := stubs.Kill{Msg: "kill"}
				res := new(stubs.StatusReport)
				conn.Go(stubs.KillBroker, req, res, nil)
				quit()
			}
		case 's':
			dstorePgm(id, conn)
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

func DistributedWorkFlow(keyPressed <-chan rune, id string) {

	// TODO: Execute all turns of the Game of Life.
	chans := make([]chan [][]byte, p.Threads)
	for i := range chans {
		chans[i] = make(chan [][]byte)
	}

	go dreportCount(id, conn)
	go dcheckKeyPressed(keyPressed, id)

	req := stubs.PublishTask{
		ID:          id,
		GolMap:      world,
		Turns:       p.Turns,
		ImageWidth:  p.ImageWidth,
		ImageHeight: p.ImageHeight,
	}

	res := new(stubs.GolResultReport)
	conn.Call(stubs.DistributorPublish, req, res)
	newWorld = res.ResultMap
	mutex.Lock()
	world = newWorld
	mutex.Unlock()

	// TODO: Report the final state using FinalTurnCompleteEvent.
	quit()
	close(c.events)
}
