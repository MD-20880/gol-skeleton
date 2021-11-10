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
	for i := 0; i < height; i++ { // should be height then width so I did it wrong here
		for j := 0; j < width; j++ {
			count := count(world, i+startY, j, p)
			copy := world[i][j]
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
				distributeChannels.events <- CellFlipped{turns, util.Cell{X: j, Y: i}}
			}
			//if world[i][j] == 255 && (count <2 || count > 3){
			//	newWorld[i][j] = 0
			//}else if world[i][j] == 0 && count == 3{
			//	newWorld[i][j] = 255
			//}else{
			//	newWorld[i][j] = world[i][j]
			//}
		}
	}
	//fmt.Println("-------THis round end--------")
	return newWorld
}

// used to count the surroundings
func count(world [][]byte, y int, x int, p Params) int {
	//fmt.Println(x, y)
	count := 0
	for i := -1; i < 2; i++ {
		for j := -1; j < 2; j++ {
			if world[(y+i+p.ImageHeight)%p.ImageHeight][(x+j+p.ImageWidth)%p.ImageWidth] == 255 {
				count += 1
			}
			// the version I used, unfinished, also got completely destroyed by sion's method
			/* if x == 0 && y != 0 && y != 15{
				if world[p.ImageWidth - 1][y + j] == 255 {
					count += 1
				}
			} else if x == p.ImageWidth - 1 && y != 0 && y != 15 {
				if world[0][y + j] == 255 {
					count += 1
				}
			}
			if y == 0 && x != 0 && x != 15 {
				if world[x + j][p.ImageHeight - 1] == 255 {
					count += 1
				}
			} else if y == p.ImageHeight - 1 && x != 0 && x != 15 {
				if world[x + j][0] == 255 {
					count += 1
				}
			}
			if !(x == 0 || y == 0 || x == p.ImageWidth - 1 || y == p.ImageHeight - 1) {
				if world[x + i][y + j] == 255 {
					count += 1
					//fmt.Println("shabi")
				}
			}*/
		}
	}
	if world[y][x] == 255 {
		count--
	}
	return count
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	container := make([]util.Cell, 0)
	//count := 0
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			if world[i][j] == 255 {
				container = append(container, util.Cell{X: j, Y: i}) // had no key here before
				//container[count] = util.Cell{j, i}
				//count++
			}
		}
	}
	return container
}
