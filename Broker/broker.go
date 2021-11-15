package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	BrokerService "uk.ac.bris.cs/gameoflife/Broker/src"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type Broker struct {
}

func (b *Broker) HandleTask(req stubs.PublishTask, res *stubs.GolResultReport) (err error) {
	fmt.Println("Receive Request")
	BrokerService.HandleTask(req, res)
	return

}

func (b *Broker) Subscribe(req stubs.Subscribe, res *stubs.StatusReport) (err error) {
	fmt.Println("Receve Subscribe")
	BrokerService.Subscribe(req, res)
	return
}

func main() {
	BrokerService.Topics = map[string]chan stubs.Work{}
	BrokerService.Buffers = map[string]chan *stubs.GolResultReport{}
	BrokerService.Subscribers = make([]*rpc.Client, 0)
	BrokerService.Topics["1"] = make(chan stubs.Work)
	BrokerService.Buffers["1"] = make(chan *stubs.GolResultReport)

	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Broker{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)

}
