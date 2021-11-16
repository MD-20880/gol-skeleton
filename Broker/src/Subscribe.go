package BrokerService

import (
	"fmt"
	"net/rpc"
	"os"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var ()

func SubscriberLoop(req stubs.Subscribe, work chan stubs.Work, workResultchan chan *stubs.GolResultReport) {
	conn, e := rpc.Dial("tcp", req.WorkerAddr)
	if e != nil {
		os.Exit(2)
	}
	id := len(Subscribers)
	Subscribers = append(Subscribers, conn)
	for {
		//build connection
		currentWork := <-work
		workResult, err := working(conn, currentWork, req.Callback)
		if err != nil {
			conn.Close()
			Subscribers = append(Subscribers[:id], Subscribers[id+1:]...)
			work <- currentWork
			break
		}
		workResultchan <- workResult
	}
}

func Subscribe(req stubs.Subscribe, res *stubs.StatusReport) (err error) {
	go SubscriberLoop(req, Topics["1"], Buffers["1"])
	res.Msg = "Get It"
	return
}

func working(conn *rpc.Client, work stubs.Work, callback string) (res *stubs.GolResultReport, err error) {
	fmt.Println(work.StartX)
	response := new(stubs.GolResultReport)
	err = conn.Call(callback, work, response)
	if err != nil {
		return
	}
	return response, nil
}
