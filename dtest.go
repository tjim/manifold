package main

// test program for Delaunay triangularization

import (
	. "./delaunay"
	. "./quadedge"
	"math/rand"
)

func main() {
	bigTri := Ngon(3, 1e4)
	for i := 0; i < 10000; i++ {
		var x, y float64
		for {
			x, y = rand.Float64(), rand.Float64() // in [0,1)
			x, y = x-0.5, y-0.5                   // in [-0.5, 0.5)
			if x*x+y*y < 0.5*0.5 {                // in unit circle
				x, y = x*200.0, y*200.0
				x, y = x+150.0, y+105.0
				break
			}
		}
		InsertSite(&Point2D{x, y}, bigTri)
	}
	Draw(bigTri)
}
