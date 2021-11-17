package main

import (
	"flag"
	"fmt"
	"github.com/ChrisGora/semaphore"
	"net"
	"net/rpc"
	BrokerService "uk.ac.bris.cs/gameoflife/Broker/src"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type Broker struct {
}

func (b *Broker) HandleTask(req stubs.PublishTask, res *stubs.GolResultReport) (err error) {
	fmt.Println("Receive Request")
	id := BrokerService.IdGenerator()
	BrokerService.Topics[id] = make(chan stubs.Work)
	BrokerService.Buffers[id] = make(chan *stubs.GolResultReport)
	BrokerService.WorkSemaList[id] = semaphore.Init(999, 0)
	BrokerService.HandleTask(req, res, id)

	return

}

func (b *Broker) Subscribe(req stubs.Subscribe, res *stubs.StatusReport) (err error) {
	fmt.Println("Receve Subscribe")
	BrokerService.Subscribe(req, res)
	res.Msg = "Got it"
	return
}

func main() {
	BrokerService.Topics = map[string]chan stubs.Work{}
	BrokerService.Buffers = map[string]chan *stubs.GolResultReport{}
	BrokerService.WorkSemaList = map[string]semaphore.Semaphore{}
	BrokerService.Subscribers = make([]*rpc.Client, 0)
	BrokerService.WorkChan = make(chan stubs.Work)

	go BrokerService.WorkDistributor()
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Broker{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)

}
