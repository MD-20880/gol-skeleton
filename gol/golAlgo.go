package gol

import "uk.ac.bris.cs/gameoflife/util"

func calculateNextState(p Params, world [][]byte, startY, endY, startX, endX int, channel chan [][]byte) {
	worldCopy := checkRule(p, world, startY, endY, startX, endX)
	channel <- worldCopy
}

func createNewWorld(height, width int) [][]byte {
	newWorld := make([][]byte, height)
	for v := range newWorld {
		newWorld[v] = make([]byte, width)
	}
	return newWorld
}

func checkRule(p Params, world [][]byte, startY, endY, startX, endX int) [][]byte {
	height := endY - startY
	width := endX - startX
	newWorld := createNewWorld(height, width)
	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			count := count(world, i+startY, j, p)
			copy := world[i+startY][j]
			if world[i+startY][j] == 255 && count < 2 {
				newWorld[i][j] = 0
			} else if world[i+startY][j] == 255 && (count == 2 || count == 3) {
				newWorld[i][j] = 255
			} else if world[i+startY][j] == 255 && count > 3 {
				newWorld[i][j] = 0
			} else if world[i+startY][j] == 0 && count == 3 {
				newWorld[i][j] = 255
			}
			if copy != newWorld[i][j] {
				distributeChannels.events <- CellFlipped{turns, util.Cell{X: j, Y: i + startY}}
			}
		}
	}
	return newWorld
}

// used to count the surroundings
func count(world [][]byte, y int, x int, p Params) int {
	count := 0
	for i := -1; i < 2; i++ {
		for j := -1; j < 2; j++ {
			if world[(y+i+p.ImageHeight)%p.ImageHeight][(x+j+p.ImageWidth)%p.ImageWidth] == 255 {
				count += 1
			}
		}
	}
	if world[y][x] == 255 {
		count--
	}
	return count
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	container := make([]util.Cell, 0)
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			if world[i][j] == 255 {
				container = append(container, util.Cell{X: j, Y: i})
			}
		}
	}
	return container
}
