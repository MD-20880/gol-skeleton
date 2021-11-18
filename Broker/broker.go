package main

import (
	"flag"
	"fmt"
	"github.com/ChrisGora/semaphore"
	"net"
	"net/rpc"
	"sync"
	BrokerService "uk.ac.bris.cs/gameoflife/Broker/src"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type Broker struct {
}

func (b *Broker) HandleTask(req stubs.PublishTask, res *stubs.GolResultReport) (err error) {
	fmt.Println("Receive Request")
	id := BrokerService.IdGenerator()
	//Initialize Topics
	BrokerService.TopicsMx.Lock()
	BrokerService.Topics[id] = make(chan stubs.Work, 1)
	BrokerService.TopicsMx.Unlock()

	//Initialize Buffers
	BrokerService.BufferMx.Lock()
	BrokerService.Buffers[id] = make(chan *stubs.GolResultReport, 1)
	BrokerService.BufferMx.Unlock()

	//Initialize WorkSemaList
	BrokerService.WorkSemaListMx.Lock()
	BrokerService.WorkSemaList[id] = semaphore.Init(1, 0)
	BrokerService.WorkSemaListMx.Unlock()

	//Start Handler
	BrokerService.HandleTask(req, res, id)
	return

}

func (b *Broker) Subscribe(req stubs.Subscribe, res *stubs.StatusReport) (err error) {
	fmt.Println("Receve Subscribe")
	BrokerService.Subscribe(req, res)
	res.Msg = "Got it"
	return
}

//Broker initialization
func initializeBroker() {
	BrokerService.Topics = map[string]chan stubs.Work{}
	BrokerService.TopicsMx = sync.RWMutex{}

	BrokerService.Buffers = map[string]chan *stubs.GolResultReport{}
	BrokerService.BufferMx = sync.RWMutex{}

	BrokerService.WorkSemaList = map[string]semaphore.Semaphore{}
	BrokerService.WorkSemaListMx = sync.RWMutex{}

	BrokerService.Subscribers = map[string]*rpc.Client{}

	BrokerService.WorkChan = make(chan stubs.Work, 1)

	BrokerService.WorkSema = semaphore.Init(999, 0)

}

func main() {

	initializeBroker()

	go BrokerService.WorkDistributor()

	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Broker{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)

}
