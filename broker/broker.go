package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

// could be problemetic when dealing with tests
var GolHandler string
var GlobalClients []*rpc.Client
var World [][]byte
var TotalTurns int
var ImageWidth int
var ImageHeight int
var DistributorClient *rpc.Client
var CompletedTurns int
var Mutex = &sync.Mutex{}
var pauseChan = make(chan bool)

func createChannels(length int) []chan [][]byte {
	channels := make([]chan [][]byte, length)
	for i := range channels {
		channels[i] = make(chan [][]byte)
	}
	return channels
}

func combine(channels []chan [][]byte, length int) [][]byte {
	var ultimateSlice [][]uint8
	for i := 0; i < length; i++ {
		v := <-channels[i]
		ultimateSlice = append(ultimateSlice, v...)
	}
	return ultimateSlice
}

func calling(i int, req stubs.BrokerRequest, channel chan [][]byte) {
	res := new(stubs.Response)
	GlobalClients[i].Call(GolHandler, req, res)
	channel <- res.World
}

// subscribe needs to dial the address
func subscribe(req stubs.Subscription, res *stubs.StatusReport) {
	client, err := rpc.Dial("tcp", req.WorkerAddress)
	GlobalClients = append(GlobalClients, client)
	if err != nil {
		fmt.Println(err)
	}
	GolHandler = req.Callback
	res.Message = "connection successful"
}

func publish(req stubs.Request, res *stubs.Response) {
	World = req.World
	TotalTurns = req.Turns
	ImageWidth = req.ImageWidth
	ImageHeight = req.ImageHeight
	client, err := rpc.Dial("tcp", req.Address) // probably don't need this
	if err != nil {
		fmt.Println(err)
	}
	DistributorClient = client
	distribute(*new(stubs.StatusReport), new(stubs.StatusReport))
	res.Turns = TotalTurns
	res.World = World
	CompletedTurns = 0
}

func checkConnection(channel chan bool) {
	for {
		response := new(stubs.StatusReport)
		DistributorClient.Call(stubs.CheckShit, new(stubs.StatusReport), response)
		if response.Message == "zhu" {
			channel <- false
		} else {
			channel <- true
		}
	}
}

func checkPause(channel chan bool) {
	for {
		<-channel
		Mutex.Lock()
		<-channel
		Mutex.Unlock()

	}
}

func distribute(req stubs.StatusReport, res *stubs.StatusReport) {
	length := len(GlobalClients)
	channels := createChannels(length)
	height := ImageHeight / length
	channel := make(chan bool)
	go checkConnection(channel)
	go checkPause(pauseChan)
	for i := 0; i < TotalTurns; i++ {
		j := 0
		for j < length-1 {
			request := stubs.BrokerRequest{Turns: TotalTurns, ImageWidth: ImageWidth, StartY: j * height, EndY: (j + 1) * height, World: World}
			go calling(j, request, channels[j])
			j++
		}
		request := stubs.BrokerRequest{Turns: TotalTurns, ImageWidth: ImageWidth, StartY: j * height, EndY: ImageHeight, World: World}
		go calling(j, request, channels[j])
		if <-channel {
			break
		}
		Mutex.Lock()
		World = combine(channels, length) // i don't know if this part would work or not, seems problematic to me
		CompletedTurns++
		Mutex.Unlock()
		fmt.Println(CompletedTurns)
	}
}

func getWorld(req stubs.StatusReport, res *stubs.Response) {
	Mutex.Lock()
	res.World = World // could be problematic
	res.Turns = CompletedTurns
	Mutex.Unlock()
}

type Broker struct {
	World [][]byte
	Turns int
}

func (b *Broker) Subscribe(req stubs.Subscription, res *stubs.StatusReport) (err error) {
	subscribe(req, res)
	return
}

func (b *Broker) Publish(req stubs.Request, res *stubs.Response) (err error) {
	publish(req, res)
	return
}

func (b *Broker) Distribute(req stubs.StatusReport, res *stubs.StatusReport) (err error) {
	distribute(req, res)
	return
}

func (b *Broker) GetWorld(req stubs.StatusReport, res *stubs.Response) (err error) {
	getWorld(req, res)
	return
}

func (b *Broker) Pause(req stubs.StatusReport, res *stubs.StatusReport) (err error) {
	pauseChan <- true
	Mutex.Lock()
	res.Number = CompletedTurns
	Mutex.Unlock()
	return
}

// TODO might have problems with global variables and mutex lock
func main() {
	pAddr := flag.String("port", "8030", "port to listen on")
	flag.Parse()
	rpc.Register(&Broker{})
	GlobalClients = make([]*rpc.Client, 0)
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		fmt.Println(err)
	}
	defer DistributorClient.Close()
	defer listener.Close()
	for i := range GlobalClients {
		defer GlobalClients[i].Close()
	}
	rpc.Accept(listener)
}
