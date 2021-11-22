package BrokerService

import (
	"fmt"
	"strconv"
	"sync"
)

var Counter int
var assginMutex sync.Mutex

func errorHandler(err error) {
	fmt.Println(err)
}

func IdGenerator() (id string) {
	assginMutex.Lock()
	id = strconv.Itoa(Counter)
	Counter++
	assginMutex.Unlock()

	return
}
