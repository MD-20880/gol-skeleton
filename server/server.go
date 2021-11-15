package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type GolOperations struct{}

func (s *GolOperations) GolWorker(req stubs.Request, res *stubs.Response) (err error) {
	//worlds := createNewWorld(req.ImageHeight, req.ImageWidth)
	for i := 0; i < req.Turns; i++ {
		req.World = CalculateNextState(req, 0, req.ImageHeight, 0, req.ImageWidth)
	}
	//res.World = worlds
	res.AliveCells = CalculateAliveCells(req)
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GolOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

func CalculateNextState(request stubs.Request, startY, endY, startX, endX int) [][]byte {
	worldCopy := checkRule(request, startY, endY, startX, endX)
	return worldCopy
}

func createNewWorld(height, width int) [][]byte {
	newWorld := make([][]byte, height)
	for v := range newWorld {
		newWorld[v] = make([]byte, width)
	}
	return newWorld
}

func checkRule(request stubs.Request, startY, endY, startX, endX int) [][]byte {
	height := endY - startY
	width := endX - startX
	newWorld := createNewWorld(height, width)
	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			count := count(request, i+startY, j)
			if request.World[i+startY][j] == 255 && count < 2 {
				newWorld[i][j] = 0
			} else if request.World[i+startY][j] == 255 && (count == 2 || count == 3) {
				newWorld[i][j] = 255
			} else if request.World[i+startY][j] == 255 && count > 3 {
				newWorld[i][j] = 0
			} else if request.World[i+startY][j] == 0 && count == 3 {
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
func count(request stubs.Request, y int, x int) int {
	//fmt.Println(x, y)
	count := 0
	for i := -1; i < 2; i++ {
		for j := -1; j < 2; j++ {
			if request.World[(y+i+request.ImageHeight)%request.ImageHeight][(x+j+request.ImageWidth)%request.ImageWidth] == 255 {
				count += 1
			}
		}
	}
	if request.World[y][x] == 255 {
		count--
	}
	return count
}

func CalculateAliveCells(request stubs.Request) []util.Cell {
	container := make([]util.Cell, 0)
	//count := 0
	for i := 0; i < request.ImageWidth; i++ {
		for j := 0; j < request.ImageHeight; j++ {
			if request.World[i][j] == 255 {
				container = append(container, util.Cell{X: j, Y: i}) // had no key here before
				//container[count] = util.Cell{j, i}
				//count++
			}
		}
	}
	return container
}
