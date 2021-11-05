package stubs

var Calculate = "Worker.Calculate"

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

type Response struct {
	Result [][]byte
}
