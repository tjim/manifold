package delaunay

import (
	. "../quadedge"
	"fmt"
	"github.com/llgcode/draw2d/draw2dpdf"
	"image/color"
	"math"
)

// Geometric predicates, Lischinski p. 10
func TriArea(a, b, c *Point2D) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func InCircle(a, b, c, d *Point2D) bool {
	return (a.X*a.X+a.Y*a.Y)*TriArea(b, c, d)-
		(b.X*b.X+b.Y*b.Y)*TriArea(a, c, d)+
		(c.X*c.X+c.Y*c.Y)*TriArea(a, b, d)-
		(d.X*d.X+d.Y*d.Y)*TriArea(a, b, c) > 0
}

func Ccw(a, b, c *Point2D) bool {
	return TriArea(a, b, c) > 0
}

func RightOf(x *Point2D, e *Edge) bool {
	return Ccw(x, e.Dest(), e.Org())
}

func LeftOf(x *Point2D, e *Edge) bool {
	return Ccw(x, e.Org(), e.Dest())
}

func OnEdge(x *Point2D, e *Edge) bool {
	norm := func(a, b *Point2D) float64 {
		x := a.X - b.X
		y := a.Y - b.Y
		return math.Sqrt(x*x + y*y)
	}
	t1 := norm(x, e.Org())
	t2 := norm(x, e.Dest())
	EPS := 1e-6
	if t1 < EPS || t2 < EPS {
		// close to org or dest
		return true
	}
	t3 := norm(e.Org(), e.Dest())
	if t1 > t3 || t2 > t3 {
		// further from org or dest than org is from dest
		return false
	}
	org := e.Org()
	dest := e.Dest()
	// plug in to eqn of line from org to dest
	return math.Abs((x.Y-org.Y)*(dest.X-org.X)-(dest.Y-org.Y)*(x.X-org.X)) < EPS
}

func Locate(x *Point2D, startingEdge *Edge) *Edge {
	e := startingEdge
	for {
		if *x == *e.Org() || *x == *e.Dest() {
			return e
		} else if RightOf(x, e) {
			e = e.Sym()
		} else if !RightOf(x, e.Onext()) {
			e = e.Onext()
		} else if !RightOf(x, e.Dprev()) {
			e = e.Dprev()
		} else {
			return e
		}
	}
}

var debug bool = false

func InsertSite(x *Point2D, startingEdge *Edge) {
	if debug {
		fmt.Printf("InsertSite %f,%f\n", x.X, x.Y)
	}
	e := Locate(x, startingEdge)
	if *x == *e.Org() || *x == *e.Dest() {
		return
	} else if OnEdge(x, e) {
		e = e.Oprev()
		DeleteEdge(e.Onext())
	}
	base := MakeEdge()
	base.SetOrg(e.Org())
	base.SetDest(x)
	Splice(base, e)
	if debug {
		Draw(base)
	}
	startingEdge = base
	for {
		base = Connect(e, base.Sym())
		if debug {
			Draw(base)
		}
		e = base.Oprev()
		if *e.Lnext() == *startingEdge {
			break
		}
	}
	for {
		t := e.Oprev()
		rightof := RightOf(t.Dest(), e)
		if debug {
			incircle := InCircle(e.Org(), t.Dest(), e.Dest(), x)
			fmt.Printf("\n%v ==\nInCircle(%#v,\n         %#v,\n         %#v,\n         %#v)\n", incircle, e.Org(), t.Dest(), e.Dest(), x)
		}
		if rightof && InCircle(e.Org(), t.Dest(), e.Dest(), x) {
			Swap(e)
			if debug {
				Draw(e)
			}
			e = e.Oprev()
		} else if *e.Onext() == *startingEdge {
			return
		} else {
			e = e.Onext().Lprev()
		}
	}
}

var fileno int = 0

func nextfile() string {
	fileno++
	return fmt.Sprintf("hello%02d.pdf", fileno)
}

func Draw(e0 *Edge) {
	file := nextfile()
	dest := draw2dpdf.NewPdf("L", "mm", "A4")
	gc := draw2dpdf.NewGraphicContext(dest)
	gc.SetLineWidth(0.1)
	for _, e := range e0.Edges() {
		gc.SetStrokeColor(color.RGBA{0x00, 0x00, 0xff, 0xff})
		gc.MoveTo(e.Org().X, e.Org().Y)
		gc.LineTo(e.Dest().X, e.Dest().Y)
		gc.Stroke()
	}
	draw2dpdf.SaveToPdfFile(file, dest)
}
