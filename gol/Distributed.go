package gol

import (
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/stubs"
)

func DistributedWorkFlow(keyPressed <-chan rune, id string) {
	conn, _ := rpc.Dial("tcp", "127.0.0.1:8030")
	defer conn.Close()

	// TODO: Execute all turns of the Game of Life.
	chans := make([]chan [][]byte, p.Threads)
	for i := range chans {
		chans[i] = make(chan [][]byte)
	}

	go dreportCount(id, conn)
	go checkKeyPressed(keyPressed)

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
