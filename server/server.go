package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type GolOperations struct{}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func (s *GolOperations) GolWorker(req stubs.BrokerRequest, res *stubs.Response) (err error) {
	//worlds := createNewWorld(req.ImageHeight, req.ImageWidth)
	// not sure this part is right tho
	if req.World == nil {
		err = errors.New("no world is given")
		return
	}
	//fmt.Println(req.World)
	req.World = calculateNextState(req, 0, req.ImageWidth)

	//res.World = worlds
	res.World = req.World
	return
}

func (s *GolOperations) KillWorker(req stubs.StatusReport, res *stubs.StatusReport) (err error) {
	os.Exit(10)
	return
}

func makeCall(client rpc.Client, pAddr string) {
	ip := GetOutboundIP()
	addr := ip.String() + ":" + pAddr
	request := stubs.Subscription{Callback: "GolOperations.GolWorker", WorkerAddress: addr}
	response := new(stubs.StatusReport)
	client.Call(stubs.Subscribe, request, response)
	fmt.Println(response.Message)
}

func main() {
	pAddr := flag.String("port", "8040", "Port to listen on")
	brokerAddr := flag.String("broker", "127.0.0.1:8030", "Address of broker instance")
	flag.Parse()
	client, err := rpc.Dial("tcp", *brokerAddr)
	handleError(err)
	defer client.Close()
	rpc.Register(&GolOperations{})
	listener, err2 := net.Listen("tcp", ":"+*pAddr)
	handleError(err2)
	makeCall(*client, *pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

func calculateNextState(request stubs.BrokerRequest, startX, endX int) [][]byte {
	worldCopy := checkRule(request, startX, endX)
	return worldCopy
}

func createNewWorld(height, width int) [][]byte {
	newWorld := make([][]byte, height)
	for v := range newWorld {
		newWorld[v] = make([]byte, width)
	}
	return newWorld
}

func checkRule(req stubs.BrokerRequest, startX, endX int) [][]byte {
	height := req.EndY - req.StartY
	width := endX - startX
	newWorld := createNewWorld(height, width)
	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			count := count(req, i+req.StartY, j)
			if req.World[i+req.StartY][j] == 255 && count < 2 {
				newWorld[i][j] = 0
			} else if req.World[i+req.StartY][j] == 255 && (count == 2 || count == 3) {
				newWorld[i][j] = 255
			} else if req.World[i+req.StartY][j] == 255 && count > 3 {
				newWorld[i][j] = 0
			} else if req.World[i+req.StartY][j] == 0 && count == 3 {
				newWorld[i][j] = 255
			}
			//if world[i][j] == 255 && (count <2 || count > 3){
			//	newWorld[i][j] = 0
			//}else if world[i][j] == 0 && count == 3{
			//	newWorld[i][j] = 255
			//}else{
			//	newWorld[i][j] = world[i][j]
			//}
		}
	}
	return newWorld
}

// used to count the surroundings
func count(request stubs.BrokerRequest, y int, x int) int {
	//fmt.Println(x, y)
	count := 0
	for i := -1; i < 2; i++ {
		for j := -1; j < 2; j++ {
			if request.World[(y+i+request.ImageWidth)%request.ImageWidth][(x+j+request.ImageWidth)%request.ImageWidth] == 255 {
				count += 1
			}
		}
	}
	if request.World[y][x] == 255 {
		count--
	}
	return count
}

func handleError(err error) {
	if err != nil {
		fmt.Println("er shazi")
	}
}
