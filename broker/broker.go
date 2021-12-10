package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

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

// subscribe needs to dial the worker addresses
func subscribe(req stubs.Subscription, res *stubs.StatusReport) {
	client, err := rpc.Dial("tcp", req.WorkerAddress)
	GlobalClients = append(GlobalClients, client)
	if err != nil {
		fmt.Println(err)
	}
	GolHandler = req.Callback
	res.Message = "connection successful"
}

// publish dials distributor's address and pass the world to the distribute function
func publish(req stubs.Request, res *stubs.Response) {
	World = req.World
	TotalTurns = req.Turns
	ImageWidth = req.ImageWidth
	ImageHeight = req.ImageHeight
	client, err := rpc.Dial("tcp", req.Address)
	if err != nil {
		fmt.Println(err)
	}
	DistributorClient = client
	distribute(*new(stubs.StatusReport), new(stubs.StatusReport))
	res.Turns = TotalTurns
	res.World = World
	CompletedTurns = 0
}

// constantly checking if the connection between the client and the broken
// are down, if so break distribute function's main loop
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

// pause the operation by locking the mutex
func checkPause(channel chan bool) {
	for {
		<-channel
		Mutex.Lock()
		<-channel
		Mutex.Unlock()
	}
}

// distributing the world and prescribed task (certain slice of the world) to different servers
// and iterates through the turns given
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
		if <-channel { // break the loop if the connection is down
			break
		}
		Mutex.Lock()
		World = combine(channels, length)
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

func killBroker() {
	Mutex.Lock()
	for _, client := range GlobalClients {
		client.Call(stubs.KillWorker, new(stubs.StatusReport), new(stubs.StatusReport))
	}
	os.Exit(10)
}

type Broker struct {
	World [][]byte
	Turns int
}

// broker's rpc call methods

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

func (b *Broker) Kill(req stubs.StatusReport, res *stubs.StatusReport) (err error) {
	go killBroker()
	return
}

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
