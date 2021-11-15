package BrokerService

import (
	"fmt"
	"sync"
	"time"
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
	for turn := 0; turn < req.Turns; turn++ {
		workList = workSplit(req)
		workNum = len(workList)
		postWork(workList)
		go workSender()
		checkWork()
		res.ResultMap = calculatingWorld
		completeWorld = calculatingWorld
		fmt.Println("Update")
	}

	res.ResultMap = completeWorld
	res.CompleteTurn = req.Turns
	res.StartX = 0
	res.StartY = 0
	res.EndY = len(req.GolMap)
	res.EndX = len(req.GolMap[0])
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
	for i, work := range workList {
		if work.StartX == jobResult.StartX {
			workMutex.Lock()
			workList = append(workList[:i], workList[i+1:]...)
			workMutex.Unlock()
			break
		}
	}
}

func checkWork() {
	for workNum > 0 {
		work := <-resultChan
		receive(work)
		for i := work.StartX; i < work.EndX; i++ {
			calculatingWorld[i] = work.ResultMap[i]
		}
		workNum--
	}
}

func workSender() {
	workMutex.RLock()
	for _, work := range workList {
		Topics["1"] <- work
	}
	workMutex.RUnlock()
	time.Sleep(500 * time.Millisecond)

	for len(workList) > 0 {
		workMutex.RLock()
		for _, work := range workList {
			Topics["1"] <- work
		}
		workMutex.RUnlock()
		time.Sleep(500 * time.Millisecond)
	}
}
