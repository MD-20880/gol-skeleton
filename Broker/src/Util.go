package BrokerService

import (
	"fmt"
	"strconv"
	"sync"
)

var counter = 0
var assginMutex sync.Mutex

func errorHandler(err error) {
	fmt.Println(err)
}

func IdGenerator() (id string) {
	assginMutex.Lock()
	id = strconv.Itoa(counter)
	assginMutex.Unlock()
	return
}
