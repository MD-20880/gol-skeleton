package BrokerService

import (
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var ()

func SubscriberLoop(conn *rpc.Client, work chan stubs.Work, workResultchan chan *stubs.GolResultReport) {
	for {
		currentWork := <-work
		workResult := working(conn, currentWork)
		workResultchan <- workResult
	}
}

func Subscribe(req stubs.Subscribe, res *stubs.StatusReport) (err error) {
	conn, e := rpc.Dial("tcp", req.WorkerAddr)
	if e != nil {
		errorHandler(e)
	}
	Subscribers = append(Subscribers, conn)
	go SubscriberLoop(conn, Topics["1"], Buffers["1"])
	return
}

func working(conn *rpc.Client, work stubs.Work) (res *stubs.GolResultReport) {
	response := new(stubs.GolResultReport)
	conn.Call(stubs.WorkerCalculate, work, response)
	return response
}
