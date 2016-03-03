package quadedge

import (
	"fmt"
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
	if d == nil {
		panic("SetOrg: nil argument")
	}
	d = &Point2D{d.X, d.Y}
	e.Q[e.R].Data = d
	//	for e1 := e.Onext(); e1.Q != e.Q; e1 = e1.Onext() {
	//		e1.Q[e1.R].Data = d
	//	}
}

func (e *Edge) Dest() *Point2D {
	return e.Sym().Org()
}

func (e *Edge) SetDest(d *Point2D) {
	e.Sym().SetOrg(d)
}

// regular polygon with n sides of length sideLength
func Ngon(n int, sideLength float64) *Edge {
	if n < 3 {
		return nil
	}
	radius := (sideLength / 2) / math.Sin(math.Pi/float64(n))
	pts := make([]*Point2D, n)
	for i := range pts {
		y, x := math.Sincos(math.Pi * float64(2*i) / float64(n))
		x, y = radius*x, radius*y
		pts[i] = &Point2D{x, y}
	}
	return Polygon(pts)
}

func Rect(a, b, c, d *Point2D) *Edge {
	return Polygon([]*Point2D{a, b, c, d})
}

func Triangle(a, b, c *Point2D) *Edge {
	return Polygon([]*Point2D{a, b, c})
}

func Polygon(pts []*Point2D) *Edge {
	n := len(pts)
	if n < 3 {
		return nil
	}

	e0 := MakeEdge()
	e0.SetOrg(pts[0])
	e0.SetDest(pts[1])

	ePrev := e0
	for i := 1; i < n; i++ {
		e := MakeEdge()
		e.SetOrg(pts[i])
		e.SetDest(pts[(i+1)%n])
		Splice(ePrev.Sym(), e)
		ePrev = e
	}

	Splice(ePrev.Sym(), e0)
	return e0
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

func (e *Edge) Edges() map[int]*Edge {
	edgeSet := make(map[*QuadEdge]bool)
	edgeIndex := make(map[int]*Edge)
	edgeIndex[0] = e
	edgeSet[e.Q] = true
	inbound := func(e1 *Edge) {
		if e1 == nil {
			return
		}
		e2 := e1.Onext()
		for e2 != nil && *e1 != *e2 {
			if edgeSet[e2.Q] == false { // have not seen this edge yet
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

func BoundingBox(e *Edge) (small, big *Point2D) {
	if e == nil || e.Org() == nil || e.Dest() == nil {
		return
	}
	small = &Point2D{e.Org().X, e.Org().Y}
	big = &Point2D{e.Org().X, e.Org().Y}
	min := func(a, b, c float64) float64 {
		switch {
		case a <= b && a <= c:
			return a
		case b <= a && b <= c:
			return b
		default:
			return c
		}
	}
	max := func(a, b, c float64) float64 {
		switch {
		case a >= b && a >= c:
			return a
		case b >= a && b >= c:
			return b
		default:
			return c
		}
	}
	for _, e1 := range e.Edges() {
		small.X = min(small.X, e1.Org().X, e1.Dest().X)
		small.Y = min(small.Y, e1.Org().Y, e1.Dest().Y)
		big.X = max(big.X, e1.Org().X, e1.Dest().X)
		big.Y = max(big.Y, e1.Org().Y, e1.Dest().Y)
	}
	return
}
