package stubs

var GolHandler = "GolOperations.GolWorker" // somehow has to declare like this

type Request struct {
	Turns       int
	ImageWidth  int
	ImageHeight int
	World       [][]byte
}

type Response struct {
	World [][]byte
	// AliveCells []util.Cell
}
