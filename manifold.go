package main

import (
	"fmt"
	. "./quadedge"
	"github.com/llgcode/draw2d/draw2dpdf"
	"image/color"
	"math"
)

var fileno int = 0

func nextfile() string {
	fileno++
	return fmt.Sprintf("hello%02d.pdf", fileno)
}

func debugDraw(e0 *Edge, e1 *Edge) {
	file := nextfile()
	dest := draw2dpdf.NewPdf("L", "mm", "A4")
	gc := draw2dpdf.NewGraphicContext(dest)
	ox, oy := 100.0, 100.0 // put origin at (100,100)
	//	small, _ := BoundingBox(e0)
	//	dx, dy := ox-small.X, oy-small.Y
	dx, dy := ox, oy
	gc.SetLineWidth(5)
	gc.SetStrokeColor(color.RGBA{0x00, 0x00, 0x00, 0xff})
	gc.MoveTo(ox, oy)
	gc.LineTo(ox, oy)
	gc.Stroke()
	gc.SetLineWidth(0.1)
	for i, e := range e0.Edges() {
		if i == 0 {
			gc.SetLineWidth(3)
			gc.SetStrokeColor(color.RGBA{0x00, 0xff, 0x00, 0xff})
			gc.MoveTo(e.Org().X+dx, e.Org().Y+dy)
			gc.LineTo(e.Org().X+dx, e.Org().Y+dy)
			gc.Stroke()
			gc.SetLineWidth(0.1)
			gc.SetStrokeColor(color.RGBA{0xff, 0x00, 0x00, 0xff})
		} else {
			gc.SetStrokeColor(color.RGBA{0x00, 0x00, 0xff, 0xff})
		}
		gc.MoveTo(e.Org().X+dx, e.Org().Y+dy)
		gc.LineTo(e.Dest().X+dx, e.Dest().Y+dy)
		gc.Stroke()
	}
	if e1 != nil {
		for i, e := range e1.Edges() {
			if i == 0 {
				gc.SetLineWidth(3)
				gc.SetStrokeColor(color.RGBA{0x00, 0xff, 0xff, 0xff})
				gc.MoveTo(e.Org().X+dx, e.Org().Y+dy)
				gc.LineTo(e.Org().X+dx, e.Org().Y+dy)
				gc.Stroke()
				gc.SetLineWidth(0.1)
				gc.SetStrokeColor(color.RGBA{0xff, 0x00, 0x00, 0xff})
			} else {
				gc.SetStrokeColor(color.RGBA{0x00, 0x00, 0xff, 0xff})
			}
			gc.MoveTo(e.Org().X+dx, e.Org().Y+dy)
			gc.LineTo(e.Dest().X+dx, e.Dest().Y+dy)
			gc.Stroke()
		}
	}
	draw2dpdf.SaveToPdfFile(file, dest)
}

func main() {

	poly1 := Ngon(3, 30)
	//	rotate(poly1, math.Pi/8)
	//	small, _ := BoundingBox(poly1)
	//	translate(poly1, -small.X, -small.Y)
	//	for _, e := range poly1.Edges() {
	//		e.Print()
	//	}
	//	draw(poly1, nextfile())
	poly2 := Ngon(5, 20)
	//	debugDraw(poly1, poly2)
	//	rotate(poly1, math.Pi/8)
	//	rotate(poly2, math.Pi/8)
	//	debugDraw(poly1, poly2)
	//
	attach(poly1, poly2)
}

func edgeLength(e *Edge) float64 {
	if e == nil {
		return 0.0
	}
	dx := e.Dest().X - e.Org().X
	dy := e.Dest().Y - e.Org().Y
	return math.Sqrt(dx*dx + dy*dy)
}

func edgeRadians(e *Edge) float64 {
	dx := e.Dest().X - e.Org().X
	dy := e.Dest().Y - e.Org().Y
	return math.Atan2(dy, dx)
}

func scale(e0 *Edge, sf float64) {
	for _, e := range e0.Edges() {
		e.SetOrg(&Point2D{sf * e.Org().X, sf * e.Org().Y})
		e = e.Sym()
		e.SetOrg(&Point2D{sf * e.Org().X, sf * e.Org().Y})
	}
}

func rotate(e0 *Edge, rad float64) {
	rotatePoint := func(p *Point2D) *Point2D {
		angle := math.Atan2(p.Y, p.X)
		distance := math.Sqrt(p.X*p.X + p.Y*p.Y)
		angle2 := angle + rad
		y, x := math.Sincos(angle2)
		x, y = distance*x, distance*y
		return &Point2D{x, y}
	}
	for _, e := range e0.Edges() {
		e.SetOrg(rotatePoint(e.Org()))
		e = e.Sym()
		e.SetOrg(rotatePoint(e.Org()))
	}
}

func translate(e0 *Edge, dx, dy float64) {
	for _, e := range e0.Edges() {
		e.SetOrg(&Point2D{dx + e.Org().X, dy + e.Org().Y})
		e = e.Sym()
		e.SetOrg(&Point2D{dx + e.Org().X, dy + e.Org().Y})
	}
}

func attach(e1, e2 *Edge) {
	debugDraw(e1, e2)
	e2.Print()
	l1 := edgeLength(e1)
	l2 := edgeLength(e2)
	if l1 == 0.0 || l2 == 0.0 {
		return
	}
	sf := l1 / l2
	translate(e2, -e2.Org().X, -e2.Org().Y) // bring origin of e2 to absolute origin (0,0)
	debugDraw(e1, e2)
	e2.Print()
	scale(e2, sf)
	debugDraw(e1, e2)
	e2.Print()
	rotate(e2, edgeRadians(e1)-edgeRadians(e2)+math.Pi)
	debugDraw(e1, e2)
	e2.Print()
	translate(e2, e1.Dest().X, e1.Dest().Y)
	debugDraw(e1, e2)
	e2.Print()
	Splice(e1.Oprev(), e2.Sym())
	Splice(e1.Sym(), e2.Oprev())
	DeleteEdge(e2)
	debugDraw(e1, nil)
}
