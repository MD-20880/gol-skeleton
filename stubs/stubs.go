package stubs

var GolHandler = "GolOperations.GolWorker"
var KillWorker = "GolOperations.KillWorker"
var Subscribe = "Broker.Subscribe"
var Publish = "Broker.Publish"
var Distribute = "Broker.Distribute"
var GetWorld = "Broker.GetWorld"
var CheckShit = "Distributor.CheckConnection"
var Pause = "Broker.Pause"
var Kill = "Broker.Kill"

// Request that get sent down from the distributor to the broker
type Request struct {
	Turns       int
	ImageWidth  int
	ImageHeight int
	World       [][]byte
	Address     string
}

// BrokerRequest that are sent to the servers
type BrokerRequest struct {
	Turns      int
	ImageWidth int
	StartY     int
	EndY       int
	World      [][]byte
}

// Response that are sent back to the distributor
type Response struct {
	World [][]byte
	Turns int
}

type Subscription struct {
	Callback      string
	WorkerAddress string
}

// StatusReport placeholder
type StatusReport struct {
	Message string
	Number  int
}
