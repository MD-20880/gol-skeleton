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
		// fmt.Println(v)
		ultimateSlice = append(ultimateSlice, v...)
	}
	// fmt.Println(ultimateSlice)
	return ultimateSlice
}

func calling(i int, req stubs.BrokerRequest, channel chan [][]byte) {
	res := new(stubs.Response)
	GlobalClients[0].Call(stubs.GolHandler, req, res)
	//fmt.Println(res.World)
	channel <- res.World
}

// subscribe needs to dial the address
func subscribe(req stubs.Subscription, res *stubs.StatusReport) {
	client, err := rpc.Dial("tcp", req.WorkerAddress)
	GlobalClients = append(GlobalClients, client)
	if err != nil {
		fmt.Println(err)
	}
	//defer client.Close()
	GolHandler = req.Callback
	res.Message = "connection successful"
}

func publish(req stubs.Request, res *stubs.Response) {
	World = req.World
	TotalTurns = req.Turns
	ImageWidth = req.ImageWidth
	ImageHeight = req.ImageHeight
	//client, err := rpc.Dial("tcp", req.Address) // probably don't need this
	//if err != nil {
	//	fmt.Println(err)
	//}
	//DistributorClient = client
	// fmt.Println(World) okay here
	distribute(*new(stubs.StatusReport), new(stubs.StatusReport))
	res.Turns = TotalTurns
	res.World = World
	// fmt.Println(World)
}

func distribute(req stubs.StatusReport, res *stubs.StatusReport) {
	length := len(GlobalClients)
	channels := createChannels(length)
	height := ImageHeight / length
	for i := 0; i < TotalTurns; i++ {
		j := 0
		for j < length-1 {
			request := stubs.BrokerRequest{Turns: TotalTurns, ImageWidth: ImageWidth, StartY: j * height, EndY: (j + 1) * height, World: World}
			go calling(j, request, channels[j])
			j++
		}
		request := stubs.BrokerRequest{Turns: TotalTurns, ImageWidth: ImageWidth, StartY: j * height, EndY: ImageHeight, World: World}
		go calling(j, request, channels[j])
		World = combine(channels, length) // i don't know if this part would work or not, seems problematic to me
		CompletedTurns++
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
	defer listener.Close()
	for i := range GlobalClients {
		defer GlobalClients[i].Close()
	}
	rpc.Accept(listener)
}
