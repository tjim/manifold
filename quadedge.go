package main

import (
	"fmt"
	"github.com/llgcode/draw2d/draw2dimg"
	"image"
	"image/color"
	"math"
)

/* Quad Edge data structure from section 4.1 (for when a single orientation is sufficient) of

   Primitives for the Manipulation of General Subdivisions and the Computation of Voronoi Diagrams
   Leonidas Guibas and Jorge Stolfi
   ACM Transactions on Graphics, Vol. 4, No. 2, April 1985, Pages 74-123.
*/
type Point2D struct {
	X, Y float64
}
type EdgePart struct {
	Data *Point2D
	Next *Edge
}
type QuadEdge [4]EdgePart
type Edge struct {
	Q *QuadEdge
	R int // Invariant: 0 <= R < 4
}

// Primitive algebraic operations
func (e *Edge) Rot() *Edge {
	return &Edge{e.Q, (e.R + 1) % 4}
}

func (e *Edge) Onext() *Edge {
	return e.Q[e.R].Next
}

// Derived algebraic operations
// (Most of the derived operations could be implemented more efficiently but we don't bother, e.g., see InvRot().)
func (e *Edge) InvRot() *Edge {
	return e.Rot().Rot().Rot()
	// return &Edge{e.Q, (e.R + 4 - 1) % 4} // careful with modulo on negative numbers, invariant is 0<=e.R<4
}

func (e *Edge) Oprev() *Edge {
	return e.Rot().Onext().Rot()
}

func (e *Edge) Sym() *Edge {
	return e.Rot().Rot()
}

func (e *Edge) Dnext() *Edge {
	return e.Sym().Onext().Sym()
}

func (e *Edge) Dprev() *Edge {
	return e.InvRot().Onext().InvRot()
}

func (e *Edge) Lnext() *Edge {
	return e.InvRot().Onext().Rot()
}

func (e *Edge) Lprev() *Edge {
	return e.Onext().Sym()
}

func (e *Edge) Rnext() *Edge {
	return e.Rot().Onext().InvRot()
}

func (e *Edge) Rprev() *Edge {
	return e.Sym().Onext()
}

// Basic topological operators, p. 96
func MakeEdge() *Edge {
	var Q QuadEdge = [4]EdgePart{}
	Q[0].Next = &Edge{&Q, 0}
	Q[1].Next = &Edge{&Q, 3}
	Q[2].Next = &Edge{&Q, 2}
	Q[3].Next = &Edge{&Q, 1}
	return &Edge{&Q, 0}
}

func Splice(a, b *Edge) {
	alpha := a.Onext().Rot()
	beta := b.Onext().Rot()
	a.Q[a.R].Next, b.Q[b.R].Next = b.Onext(), a.Onext()
	alpha.Q[alpha.R].Next, beta.Q[beta.R].Next = beta.Onext(), alpha.Onext()
}

// Getters and setters for geometric data
// Note that these are the "Org" and "Dest" of Section 6, p. 103,
// they are not rings of edges as in the rest of the paper
func (e *Edge) Org() *Point2D {
	return e.Q[e.R].Data
}

func (e *Edge) SetOrg(d *Point2D) {
	e.Q[e.R].Data = d
	for e1 := e.Onext(); e1.Q != e.Q; e1 = e1.Onext() {
		e1.Q[e1.R].Data = d
	}
}

func (e *Edge) Dest() *Point2D {
	return e.Sym().Org()
}

func (e *Edge) SetDest(d *Point2D) {
	e.Sym().SetOrg(d)
}

// Derived topological operators, p. 103
func Connect(a, b *Edge) *Edge {
	e := MakeEdge()
	e.SetOrg(a.Dest())
	e.SetDest(b.Org())
	Splice(e, a.Lnext())
	Splice(e.Sym(), b)
	return e
}

func DeleteEdge(e *Edge) {
	Splice(e, e.Oprev())
	Splice(e.Sym(), e.Sym().Oprev())
}

func Swap(e *Edge) {
	a := e.Oprev()
	b := e.Sym().Oprev()
	Splice(e, a)
	Splice(e.Sym(), b)
	Splice(e, a.Lnext())
	Splice(e.Sym(), b.Lnext())
	e.SetOrg(a.Dest())
	e.SetDest(b.Dest())
}

// Geometric predicates, Lischinski p. 10
func TriArea(a, b, c *Point2D) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func InCircle(a, b, c, d *Point2D) bool {
	return (a.X*a.X+a.Y*a.Y)*TriArea(b, c, d)-
		(b.X*b.X+b.Y*b.Y)*TriArea(a, c, d)-
		(c.X*c.X+c.Y*c.Y)*TriArea(a, b, d)+
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
		if x == e.Org() || x == e.Dest() {
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

func InsertSite(x *Point2D, startingEdge *Edge) {
	e := Locate(x, startingEdge)
	if x == e.Org() || x == e.Dest() {
		return
	} else if OnEdge(x, e) {
		e = e.Oprev()
		DeleteEdge(e.Onext())
	}
	base := MakeEdge()
	base.SetOrg(e.Org())
	base.SetDest(x)
	Splice(base, e)
	startingEdge = base
	for {
		base = Connect(e, base.Sym())
		e = base.Oprev()
		if e.Lnext() == startingEdge {
			break
		}
	}
	for {
		t := e.Oprev()
		if RightOf(t.Dest(), e) && InCircle(e.Org(), t.Dest(), e.Dest(), x) {
			Swap(e)
			e = e.Oprev()
		} else if e.Onext() == startingEdge {
			return
		} else {
			e = e.Onext().Lprev()
		}
	}
}

func (e *Edge) Edges() map[int]*Edge {
	fmt.Printf("Edges\n")
	//	e.Print()
	edgeSet := make(map[*QuadEdge]bool)
	edgeIndex := make(map[int]*Edge)
	edgeIndex[0] = e
	fmt.Printf("Adding %v\n", e.Q)
	edgeSet[e.Q] = true
	inbound := func(e1 *Edge) {
		if e1 == nil {
			return
		}
		e2 := e1.Onext()
		for e2 != nil && *e1 != *e2 {
			if edgeSet[e2.Q] == false { // have not seen this edge yet
				fmt.Printf("Adding %v\n", e2.Q)
				edgeSet[e2.Q] = true
				edgeIndex[len(edgeIndex)] = e2.Sym()
			}
			e2 = e2.Onext()
		}
	}
	inbound(e.Sym())
	for i := 0; i < len(edgeSet); i++ {
		ei := edgeIndex[i]
		inbound(ei)
	}
	return edgeIndex
}

func (e *Edge) Print() {
	o := e.Org()
	d := e.Dest()
	switch {
	case o == nil && d == nil:
		fmt.Printf("nil -> nil\n")
	case o == nil:
		fmt.Printf("nil -> %f,%f\n", d.X, d.Y)
	case d == nil:
		fmt.Printf("%f,%f -> nil\n", o.X, o.Y)
	default:
		fmt.Printf("%f,%f -> %f,%f\n", e.Org().X, e.Org().Y, e.Dest().X, e.Dest().Y)
	}
}

func PrintEdges(m map[int]*Edge) {
	for i := 0; i < len(m); i++ {
		e := m[i]
		e.Print()
	}
}

var indexQ map[*QuadEdge]int

func (e *Edge) IndexQ() int {
	if indexQ[e.Q] == 0 {
		indexQ[e.Q] = len(indexQ) + 1
	}
	return indexQ[e.Q]
}

func (e *Edge) DebugString() string {
	return fmt.Sprintf("%d:%d", e.IndexQ(), e.R)
}

func (e *Edge) DotPrint() {
	edges := e.Edges()
	indexQ = make(map[*QuadEdge]int)
	fmt.Printf("digraph g {\n\tnode [shape=record]\n")
	for _, ei := range edges {
		fmt.Printf("\t%d [label=\"{|<0>|}|{<1>||<3>}|{|<2>|}\"]\n", ei.IndexQ())
	}
	for _, ei := range edges {
		fmt.Printf("\t%s -> %s\n", ei.DebugString(), ei.Onext().DebugString())
		ei = ei.Rot()
		fmt.Printf("\t%s -> %s\n", ei.DebugString(), ei.Onext().DebugString())
		ei = ei.Rot()
		fmt.Printf("\t%s -> %s\n", ei.DebugString(), ei.Onext().DebugString())
		ei = ei.Rot()
		fmt.Printf("\t%s -> %s\n", ei.DebugString(), ei.Onext().DebugString())
	}
	fmt.Printf("}\n")
}

func main() {
	fmt.Printf("Hello world\n")
	e1 := MakeEdge()
	e1.SetOrg(&Point2D{3, 5})
	fmt.Printf("%v\n", *e1)
	fmt.Printf("%v\n", e1.Q)
	e2 := MakeEdge()
	e2.SetOrg(&Point2D{100, 100})
	Splice(e1, e2)
	e1.Print()
	e2.Print()
	//	e2.SetOrg(&Point2D{100, 100})
	//	e2.Print()
	//	Connect(e1, e2)
	//	e3.Print()
	PrintEdges(e1.Edges())

	fmt.Printf("Hello world\n")
	a := MakeEdge().Rot()
	a.DotPrint()
	b := MakeEdge().Rot()
	c := MakeEdge().Rot()
	a.SetOrg(&Point2D{1, 1})
	b.SetOrg(&Point2D{2, 2})
	c.SetOrg(&Point2D{3, 3})
	a.Print()
	b.Print()
	c.Print()
	Splice(a, b)
	Splice(b, c)
	a.SetOrg(&Point2D{1, 1})
	b.SetOrg(&Point2D{2, 2})
	c.SetOrg(&Point2D{3, 3})
	fmt.Printf("a: %v\n", a.Q)
	fmt.Printf("b: %v\n", b.Q)
	fmt.Printf("c: %v\n", c.Q)
	PrintEdges(a.Edges())
	a.DotPrint()
	fmt.Printf("Hello world\n")

	i := 4
	fmt.Printf("i: %x, &i: %x\n", i, &i)
	fmt.Printf("&a: %x\n", &a.Q)
	q := a.Q
	fmt.Printf("q: %p\n", q)

	// Initialize the graphic context on an RGBA image
	dest := image.NewRGBA(image.Rect(0, 0, 297, 210.0))
	gc := draw2dimg.NewGraphicContext(dest)

	// Set some properties
	gc.SetFillColor(color.RGBA{0x44, 0xff, 0x44, 0xff})
	gc.SetStrokeColor(color.RGBA{0x44, 0x44, 0x44, 0xff})
	gc.SetLineWidth(5)

	// Draw a closed shape
	gc.MoveTo(10, 10) // should always be called first for a new path
	gc.LineTo(100, 50)
	gc.QuadCurveTo(100, 10, 10, 10)
	gc.Close()
	gc.FillStroke()

	// Save to file
	draw2dimg.SaveToPngFile("hello.png", dest)
}
