package main

import (
	"fmt"
	"os"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
)

func BenchmarkGol(b *testing.B) {
	os.Stdout = nil
	p := gol.Params{ImageWidth: 5120, ImageHeight: 5120, Turns: 50}

	//Height ,Width ,Turns ,#Workers ,Threads on each worker
	testName := fmt.Sprintf("%dx%dx%dx%dx%d", p.ImageWidth, p.ImageHeight, p.Turns, 4, 2)
	b.Run(testName, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			events := make(chan gol.Event)
			go gol.Run(p, events, nil)
		LOOP:
			for event := range events {
				switch event.(type) {
				case gol.FinalTurnComplete:
					break LOOP
				}
			}

		}

	})

	fmt.Println("BenchMark Finished ")
}
