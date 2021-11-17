package BrokerService

import (
	"fmt"
	"os"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type variables struct {
	id               string
	req              stubs.PublishTask
	res              *stubs.GolResultReport
	CompleteWorld    [][]byte
	CalculatingWorld [][]byte
	WorkList         []stubs.Work
	WorkNum          int
	ResultChan       chan *stubs.GolResultReport
}

func initVars(req stubs.PublishTask, res *stubs.GolResultReport, id string) (v variables) {
	WorkList := make([]stubs.Work, 0)
	ResultChan := Buffers[id]
	CompleteWorld := req.GolMap
	CalculatingWorld := make([][]byte, len(CompleteWorld))
	for i := 0; i < len(CompleteWorld); i++ {
		CalculatingWorld[i] = make([]byte, len(CompleteWorld[i]))
	}
	return variables{
		id:               id,
		req:              req,
		res:              res,
		CompleteWorld:    CompleteWorld,
		CalculatingWorld: CalculatingWorld,
		WorkList:         WorkList,
		WorkNum:          0,
		ResultChan:       ResultChan,
	}
}

func workSplit(v variables) []stubs.Work {
	splitResult := make([]stubs.Work, 0)
	noSubscribers := len(Subscribers)
	if noSubscribers == 0 {
		os.Exit(3)
	}
	for i := 0; i < noSubscribers-1; i++ {
		splitResult = append(splitResult, stubs.Work{
			Turns:        v.req.Turns,
			ImageWidth:   v.req.ImageWidth,
			ImageHeight:  v.req.ImageHeight,
			StartX:       i * v.req.ImageHeight / noSubscribers,
			StartY:       0,
			EndX:         (i + 1) * v.req.ImageWidth / noSubscribers,
			EndY:         v.req.ImageWidth,
			CalculateMap: v.CompleteWorld,
			Owner:        v.id,
		})
	}
	splitResult = append(splitResult, stubs.Work{
		Turns:        v.req.Turns,
		ImageWidth:   v.req.ImageWidth,
		ImageHeight:  v.req.ImageHeight,
		StartX:       (noSubscribers - 1) * v.req.ImageHeight / noSubscribers,
		StartY:       0,
		EndX:         v.req.ImageHeight,
		EndY:         v.req.ImageWidth,
		CalculateMap: v.CompleteWorld,
		Owner:        v.id,
	})
	return splitResult
}

func postWork(workList []stubs.Work, id string) {
	for _, work := range workList {

		workList = append(workList, work)
	}
}

func workSender(workList []stubs.Work, id string) {
	//WorkMutex.RLock()
	for _, work := range workList {
		WorkSemaList[id].Post()
		fmt.Printf("%s : %d\n", id, WorkSemaList[id].GetValue())
		WorkSema.Post()
		fmt.Printf("WorkSema : %d\n", WorkSema.GetValue())
		Topics[id] <- work
	}
	fmt.Println("Sending Loop End")
	//WorkMutex.RUnlock()
	//time.Sleep(500 * time.Millisecond)
	//
	//for len(WorkList) > 0 {
	//	WorkMutex.RLock()
	//	for _, work := range WorkList {
	//		Topics["1"] <- work
	//	}
	//	WorkMutex.RUnlock()
	//	time.Sleep(500 * time.Millisecond)
	//}
}

func checkWork(v variables) {
	for v.WorkNum > 0 {
		work := <-v.ResultChan
		if work.EndX > len(v.CalculatingWorld) {
			break
		}
		fmt.Println("Receiving")
		receive(work, v)
		fmt.Println("Finish receiving")
		for i := work.StartX; i < work.EndX; i++ {
			v.CalculatingWorld[i] = work.ResultMap[i-work.StartX]
		}
		v.WorkNum--
	}
}

func receive(jobResult *stubs.GolResultReport, v variables) {
	if len(v.WorkList) == 0 {
		return
	}
	for i, work := range v.WorkList {
		if work.StartX == jobResult.StartX {
			if len(v.WorkList) > 1 {
				v.WorkList = append(v.WorkList[:i], v.WorkList[i+1:]...)
			} else {
				v.WorkList = make([]stubs.Work, 0)
			}
			break
		}
	}
}

func reply(v variables) {

}

func closeHandler(id string) {
	close(Topics[id])
	delete(Topics, id)
	close(Buffers[id])
	delete(Buffers, id)
}

func checkEvent() {

}

func HandleTask(req stubs.PublishTask, res *stubs.GolResultReport, id string) (err error) {
	//Initialize variables
	v := initVars(req, res, id)
	//Task Cycle
	for turn := 0; turn < req.Turns; turn++ {
		fmt.Println("Spliting")
		//Split One big task into several small tasks
		v.WorkList = workSplit(v)
		//Record the number of work been send
		v.WorkNum = len(v.WorkList)
		fmt.Println("Posting")
		//Post Work
		postWork(v.WorkList, v.id)
		fmt.Println("Sending")
		workSender(v.WorkList, v.id)
		fmt.Println("checking")
		checkWork(v)
		//res.ResultMap = CalculatingWorld
		v.CompleteWorld = v.CalculatingWorld
		CalculatingWorld := make([][]byte, len(v.CompleteWorld))
		for i := 0; i < len(v.CompleteWorld); i++ {
			CalculatingWorld[i] = make([]byte, len(v.CompleteWorld[i]))
		}
		fmt.Println("Updated")
	}
	//Response to Request
	//reply(v)
	res.ResultMap = v.CompleteWorld
	res.CompleteTurn = v.req.Turns
	res.StartX = 0
	res.StartY = 0
	res.EndY = len(v.req.GolMap)
	res.EndX = len(v.req.GolMap[0])
	closeHandler(v.id)

	return
}
