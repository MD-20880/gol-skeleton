package BrokerService

import (
	"fmt"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var completeWorld [][]byte
var calculatingWorld [][]byte
var workList []stubs.Work
var workMutex sync.RWMutex
var workNum int
var resultChan chan *stubs.GolResultReport

func HandleTask(req stubs.PublishTask, res *stubs.GolResultReport) (err error) {
	workList = make([]stubs.Work, 0)
	resultChan = Buffers["1"]
	completeWorld = req.GolMap
	calculatingWorld = make([][]byte, len(completeWorld))
	for i := 0; i < len(completeWorld); i++ {
		calculatingWorld[i] = make([]byte, len(completeWorld[i]))
	}
	for turn := 0; turn < req.Turns; turn++ {
		fmt.Println("Spliting")
		workList = workSplit(req)
		workNum = len(workList)
		fmt.Println("Posting")
		postWork(workList)
		fmt.Println("Sending")
		workSender()
		fmt.Println("checking")
		checkWork()
		//res.ResultMap = calculatingWorld
		completeWorld = calculatingWorld
		fmt.Println("Updated")
	}

	res.ResultMap = completeWorld
	res.CompleteTurn = req.Turns
	res.StartX = 0
	res.StartY = 0
	res.EndY = len(req.GolMap)
	res.EndX = len(req.GolMap[0])

	//reinitialize
	workList = make([]stubs.Work, 0)
	resultChan = Buffers["1"]
	completeWorld = req.GolMap
	calculatingWorld = make([][]byte, len(completeWorld))
	for i := 0; i < len(completeWorld); i++ {
		calculatingWorld[i] = make([]byte, len(completeWorld[i]))
	}
	return
}

func workSplit(req stubs.PublishTask) []stubs.Work {
	splitResult := make([]stubs.Work, 0)
	noSubscribers := len(Subscribers)
	for i := 0; i < noSubscribers-1; i++ {
		splitResult = append(splitResult, stubs.Work{
			Turns:        req.Turns,
			ImageWidth:   req.ImageWidth,
			ImageHeight:  req.ImageHeight,
			StartX:       i * req.ImageHeight / noSubscribers,
			StartY:       0,
			EndX:         (i + 1) * req.ImageWidth / noSubscribers,
			EndY:         req.ImageWidth,
			CalculateMap: completeWorld,
		})
	}
	splitResult = append(splitResult, stubs.Work{
		Turns:        req.Turns,
		ImageWidth:   req.ImageWidth,
		ImageHeight:  req.ImageHeight,
		StartX:       (noSubscribers - 1) * req.ImageHeight / noSubscribers,
		StartY:       0,
		EndX:         req.ImageHeight,
		EndY:         req.ImageWidth,
		CalculateMap: completeWorld,
	})
	return splitResult
}

func postWork(workList []stubs.Work) {
	for _, work := range workList {
		workList = append(workList, work)
	}
}

func receive(jobResult *stubs.GolResultReport) {
	if len(workList) == 0 {
		return
	}
	for i, work := range workList {
		if work.StartX == jobResult.StartX {
			fmt.Println("requireing lock")
			workMutex.Lock()
			fmt.Println("mutex.lock")
			if len(workList) > 1 {
				workList = append(workList[:i], workList[i+1:]...)
			} else {
				workList = make([]stubs.Work, 0)
			}
			fmt.Println("mutex.unlock")
			workMutex.Unlock()
			break
		}
	}
}

func checkWork() {
	for workNum > 0 {
		work := <-resultChan
		fmt.Println("Receiving")
		receive(work)
		fmt.Println("Finish receiving")
		for i := work.StartX; i < work.EndX; i++ {
			fmt.Printf("Endx = %d\n", work.EndX)
			fmt.Printf("len(calculatingWorld[i] = %d\n", len(calculatingWorld))
			calculatingWorld[i] = work.ResultMap[i-work.StartX]
		}
		fmt.Println(workNum)
		workNum--
	}
}

func workSender() {
	//workMutex.RLock()
	for _, work := range workList {
		Topics["1"] <- work
	}
	//workMutex.RUnlock()
	//time.Sleep(500 * time.Millisecond)
	//
	//for len(workList) > 0 {
	//	workMutex.RLock()
	//	for _, work := range workList {
	//		Topics["1"] <- work
	//	}
	//	workMutex.RUnlock()
	//	time.Sleep(500 * time.Millisecond)
	//}
}
