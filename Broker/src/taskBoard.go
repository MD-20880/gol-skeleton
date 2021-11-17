package BrokerService

import (
	"fmt"
	"github.com/ChrisGora/semaphore"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var Subscribers []*rpc.Client
var Topics map[string]chan stubs.Work
var TopicsMx sync.RWMutex
var Buffers map[string]chan *stubs.GolResultReport

var WorkSemaList map[string]semaphore.Semaphore
var WorkSema = semaphore.Init(999, 0)

var WorkChan = make(chan stubs.Work)

func WorkDistributor() {
	for {
		fmt.Println("Requireing workSema")
		WorkSema.Wait()
		fmt.Println("Get WorkSema")
		for key := range WorkSemaList {
			sema := WorkSemaList[key]
			if sema.GetValue() == 0 {
				continue
			}
			fmt.Println("Waiting for Sema")
			sema.Wait()
			fmt.Println("Get Sema")
			work := <-Topics[key]
			WorkChan <- work
			fmt.Println("Function End")
			break
		}
	}
}
