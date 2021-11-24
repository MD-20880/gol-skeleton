package stubs

var GolHandler = "GolOperations.GolWorker" // somehow has to declare like this
var Subscribe = "Broker.Subscribe"
var Publish = "Broker.Publish"
var Distribute = "Broker.Distribute"
var GetWorld = "Broker.GetWorld"

// Request this is the same as publishRequest
// yeah, now I don't think its gonna work, it needs that method directly its not like you can
// pass it around
type Request struct {
	Turns       int
	ImageWidth  int
	ImageHeight int
	World       [][]byte
	Address     string
}

type BrokerRequest struct {
	Turns      int
	ImageWidth int
	StartY     int
	EndY       int
	World      [][]byte
}

type Response struct {
	World [][]byte
	Turns int
	// AliveCells []util.Cell
}

type Subscription struct {
	Callback      string
	WorkerAddress string // to be more specific only needs the port
}

type StatusReport struct {
	Message string
}
