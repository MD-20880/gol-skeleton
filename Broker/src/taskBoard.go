package BrokerService

import (
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var Subscribers []*rpc.Client
var Topics map[string]chan stubs.Work
var TopicsMx sync.RWMutex
var Buffers map[string]chan *stubs.GolResultReport
