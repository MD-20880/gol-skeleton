package stubs

var WorkerCalculate = "Worker.Calculate"
var DistributorPublish = "Broker.HandleTask"
var Calculate = "Worker.Calculate"
var WorkerSubscribe = "Broker.Subscribe"

type Cell struct {
	X, Y int
}

type Request struct {
	Turns        int
	ImageWidth   int
	ImageHeight  int
	StartX       int
	StartY       int
	EndX         int
	EndY         int
	CalculateMap [][]byte
}

//Rule: every task published by client.GO
// 		every result returned by client.Call

type Response struct {
	Result [][]byte
}

// Distributor -> Broker ( publish task )
type PublishTask struct {
	ID          string
	GolMap      [][]byte
	Turns       int
	ImageWidth  int
	ImageHeight int
}

type Subscribe struct {
	WorkerAddr string
	Callback   string
}

// response for Gol result request
type GolResultReport struct {
	StartX       int
	StartY       int
	EndX         int
	EndY         int
	ResultMap    [][]byte
	CompleteTurn int
}

// request for Gol result request
type Work struct {
	Turns        int
	ImageWidth   int
	ImageHeight  int
	StartX       int
	StartY       int
	EndX         int
	EndY         int
	CalculateMap [][]byte
	Owner        string
}

type StatusReport struct {
	Msg string
}

type SdlUpdate struct {
	TurnComplete int
	flipCells    []Cell
}

type RequestCurrentWorld struct {
	ID string
}

type RespondCurrentWorld struct {
	World [][]byte
	Turn  int
}
