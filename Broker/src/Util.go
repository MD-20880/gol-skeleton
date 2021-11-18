package BrokerService

import (
	"fmt"
	"strconv"
	"sync"
)

var counter = 0
var assginMutex sync.Mutex
var idCollection []string
var idCollectionMx sync.RWMutex

func errorHandler(err error) {
	fmt.Println(err)
}

func IdGenerator() (id string) {
	assginMutex.Lock()
	id = strconv.Itoa(counter)
	counter++
	assginMutex.Unlock()

	idCollectionMx.Lock()
	idCollection = append(idCollection, id)
	idCollectionMx.Unlock()
	return
}

func SubscriberIdGenerator() (id string) {
	return
}

func removeWorkId(delId string) {
	idCollectionMx.Lock()
	for i, id := range idCollection {
		if id == delId {
			idCollection = append(idCollection[:i], idCollection[i+1:]...)
			break
		}
	}
	idCollectionMx.Unlock()

}
