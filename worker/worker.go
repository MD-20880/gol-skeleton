package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"strconv"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

func calculateNextState(p Params, world [][]byte, startX int, startY int, endX int, endY int, resultChan chan [][]byte) {
	x_scan_map := [3]int{-1, 0, 1}
	y_scan_map := [3]int{-1, 0, 1}

	newWorld := make([][]byte, endX-startX)
	for i := range newWorld {
		newWorld[i] = make([]byte, endY-startY)
	}

	for i := startX; i < endX; i++ {
		for j := startY; j < endY; j++ {
			c := make(chan byte, 10)
			calculateHelper(i, j, &world, x_scan_map, y_scan_map, p, c)
			result := <-c
			//fmt.Printf("startX : %d\n",startX)
			//fmt.Printf("startY : %d\n",startY)
			newWorld[i-startX][j-startY] = result
		}
	}
	resultChan <- newWorld
}

func calculateHelper(x int, y int, oldWorld *[][]byte, xmap [3]int, ymap [3]int, p Params, c chan byte) {
	d_oldWorld := *oldWorld
	alive := 0
	check := func(x_cor int, y_cor int) int {
		if d_oldWorld[x_cor][y_cor] == 255 {
			return 1
		}
		return 0
	}

	for _, x_scan := range xmap {
		xcal := x
		if x+x_scan > p.ImageWidth-1 {
			xcal = 0
		} else if x+x_scan < 0 {
			xcal = p.ImageWidth - 1
		} else {
			xcal = xcal + x_scan
		}
		for _, y_scan := range ymap {
			if x_scan == 0 && y_scan == 0 {
				continue
			}
			if y+y_scan > p.ImageHeight-1 {
				alive += check(xcal, 0)
			} else if y+y_scan < 0 {
				alive += check(xcal, p.ImageHeight-1)
			} else {
				alive += check(xcal, y+y_scan)
			}
		}
	}

	if d_oldWorld[x][y] == 255 && (alive < 2 || alive > 3) {
		c <- 0

	} else if d_oldWorld[x][y] == 0 && alive == 3 {
		c <- 255
	} else {
		c <- d_oldWorld[x][y]
	}
}

func StartWorker(p Params, world [][]byte, startX int, startY int, endX int, endY int, resultChan chan [][]byte) {
	calculateNextState(p, world, startX, startY, endX, endY, resultChan)
}

type Worker struct {
}

func (w *Worker) Calculate(request stubs.Work, response *stubs.GolResultReport) (err error) {
	fmt.Printf("Request received\n")
	p := Params{
		Turns:       request.Turns,
		Threads:     1,
		ImageWidth:  request.ImageWidth,
		ImageHeight: request.ImageHeight,
	}
	r := request
	resultMap := make(chan [][]byte, 1)
	StartWorker(p, r.CalculateMap, r.StartX, r.StartY, r.EndX, r.EndY, resultMap)
	resultWorld := <-resultMap
	response.StartX = request.StartX
	response.EndX = request.EndX
	response.StartY = request.StartY
	response.ResultMap = resultWorld
	response.CompleteTurn = request.Turns
	response.EndY = request.EndY
	fmt.Println("Request Finish")
	return
}
func subscribeBroker(bAddr string, pAddr string) {
	conn, _ := rpc.Dial("tcp", bAddr)
	addr := "127.0.0.1:" + pAddr
	req := stubs.Subscribe{
		WorkerAddr: addr,
		Callback:   stubs.WorkerCalculate,
	}
	res := new(stubs.StatusReport)
	conn.Call(stubs.WorkerSubscribe, req, res)
	fmt.Printf(res.Msg)
	conn.Close()
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	bAddr := flag.String("broker", "127.0.0.1:8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Worker{})
	listener, err := net.Listen("tcp", ":"+*pAddr)
	for err != nil {
		result, _ := strconv.Atoi(*pAddr)
		*pAddr = strconv.Itoa(result + 10)
		listener, err = net.Listen("tcp", ":"+*pAddr)
	}
	subscribeBroker(*bAddr, *pAddr)
	defer listener.Close()
	fmt.Println("Listining ")
	rpc.Accept(listener)

}
