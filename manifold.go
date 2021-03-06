package main

import (
	. "./quadedge"
	"fmt"
	"github.com/ajstarks/svgo/float"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	//	"text/template"
	"bytes"
)

var fileno int = 0

func nextfile() string {
	fileno++
	return fmt.Sprintf("hello%02d.svg", fileno)
}

var debug bool = false

func debugDraw(e0 *Edge, e1 *Edge) {
	if !debug {
		return
	}
	file, err := os.Create(nextfile())
	if err != nil {
		panic("can't create file")
	}
	s := svg.New(file)
	s.Start(1000, 1000)
	ox, oy := 100.0, 100.0 // put origin at (100,100)
	//	small, _ := BoundingBox(e0)
	//	dx, dy := ox-small.X, oy-small.Y
	dx, dy := ox, oy
	s.Circle(ox, oy, 5, "fill:black;stroke:black")
	for i, e := range e0.Edges() {
		if i == 0 {
			s.Circle(e.Org().X+dx, e.Org().Y+dy, 3, "fill:green;stroke:none")
			s.Line(e.Org().X+dx, e.Org().Y+dy,
				e.Dest().X+dx, e.Dest().Y+dy,
				"stroke:#f00;stroke-width:1")
		} else {
			s.Line(e.Org().X+dx, e.Org().Y+dy,
				e.Dest().X+dx, e.Dest().Y+dy,
				"stroke:#00f;stroke-width:1")
		}
	}
	if e1 != nil {
		for i, e := range e1.Edges() {
			if i == 0 {
				s.Circle(e.Org().X+dx, e.Org().Y+dy, 3, "fill:blue;stroke:none")
				s.Line(e.Org().X+dx, e.Org().Y+dy,
					e.Dest().X+dx, e.Dest().Y+dy,
					"stroke:#f00;stroke-width:1")
			} else {
				s.Line(e.Org().X+dx, e.Org().Y+dy,
					e.Dest().X+dx, e.Dest().Y+dy,
					"stroke:#00f;stroke-width:1")
			}
		}
	}
	s.End()
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

func absAngle(rad float64) float64 {
	for math.Abs(rad) > math.Pi {
		if rad < 0 {
			rad = rad + 2*math.Pi
		} else {
			rad = rad - 2*math.Pi
		}
	}
	return rad / math.Pi * 180
}

func tab0(pts []*Point2D) *Edge {
	p := Polygon(pts)
	for _, e := range p.Edges() {
		if *e == *p {
			continue
		}
		tabEdge[e.Q] = true
	}
	return p
}

func traySide() *Edge {
	return Polygon([]*Point2D{{0, 0}, {100, 0}, {110, 30}, {-10, 30}})
}

func tab(e *Edge) *Edge { // a tab that pays attention to narrow angles
	epsilon := 1e-10 // a bit bigger than zero to allow for inaccuracy in calculating angles
	cwAngle := 45.0
	cwSym := absAngle(edgeRadians(cwPerimeter(e)) - edgeRadians(e.Sym()))
	if cwSym < epsilon && math.Abs(cwSym) < cwAngle {
		cwAngle = math.Abs(cwSym)
	}
	ccwAngle := 45.0
	symCcw := absAngle(edgeRadians(e.Sym()) - edgeRadians(ccwPerimeter(e)))
	if symCcw < epsilon && math.Abs(symCcw) < ccwAngle {
		ccwAngle = math.Abs(symCcw)
	}
	alphaAngle := ccwAngle
	betaAngle := cwAngle
	/* Draw a tab with left and right angles alpha and beta, with height < width/4.

	   There are two cases.

	   Case 1:

		     D-------------C              -
		    /               \             |
		   /                 \            1
		  /                   \           |
	   alpha A---------------------B beta     -

		 |--------- 4 ---------|

	   Here alpha is the angle BAD and beta is the angle ABC, and they are known.
	   If A = (0, 0) and B = (4, 0),
	   we need the coordinates of C and D:
	   C = (4 - 1/tan(beta), 1)
	   D = (1/tan(alpha), 1)

	   Case 2:

		     C              -
		    / \             |
		   /   \          h < 1
		  /     \           |
	   alpha A-------B beta     -

		 |-- 4 --|

	   Again A = (0, 0) and B = (4, 0),
	   and we need the coordinates of C.
	   Let gamma = the angle ACB = (180 - alpha - beta).
	   By the law of sines,

	       4/sin(gamma) = AC/sin(beta) = BC/sin(alpha)

	   So

	       AC = 4*sin(beta)/sin(gamma)

	   Therefore we can calculate the coordinates of C,

	       AC * cos(alpha), AC * sin(alpha)

	   or

	       4*sin(beta)/sin(gamma)*cos(alpha), 4*sin(beta)/sin(gamma)*sin(alpha)

	   To see whether we are in case 1 or two, check 4*sin(beta)/sin(gamma)*sin(alpha) < 1
	*/
	gammaAngle := 180 - alphaAngle - betaAngle
	alpha := alphaAngle / 180 * math.Pi
	beta := betaAngle / 180 * math.Pi
	gamma := gammaAngle / 180 * math.Pi
	if 4*math.Sin(beta)/math.Sin(gamma)*math.Sin(alpha) < 1 {
		// case 2
		pts := []*Point2D{{0, 0}, {4, 0}, {4 * math.Sin(beta) / math.Sin(gamma) * math.Cos(alpha), 4 * math.Sin(beta) / math.Sin(gamma) * math.Sin(alpha)}}
		return tab0(pts)
	} else {
		// case 1
		pts := []*Point2D{{0, 0}, {4, 0}, {4 - 1/math.Tan(beta), 1}, {1 / math.Tan(alpha), 1}}
		return tab0(pts)
	}
}

func halfsies(e *Edge) *Edge {
	// split the edge e into two edges:
	// A----e---->B becomes A----e1---->A'----e---->B
	// New vertex A' is halfway between A and B
	// Return the new edge e1

	prev := e.Oprev()
	Splice(e, prev)
	e1 := MakeEdge()
	Splice(e1, prev)
	Splice(e1.Sym(), e)

	org := e.Org()
	dest := e.Dest()
	mid := &Point2D{(dest.X + org.X) / 2, (dest.Y + org.Y) / 2}
	e1.SetOrg(org)
	e1.SetDest(mid)
	e.SetOrg(mid)

	return e1
}

func attach(e1, e2 *Edge) {
	debugDraw(e1, e2)
	l1 := edgeLength(e1)
	l2 := edgeLength(e2)
	if l1 == 0.0 || l2 == 0.0 {
		return
	}
	sf := l1 / l2
	translate(e2, -e2.Org().X, -e2.Org().Y) // bring origin of e2 to absolute origin (0,0)
	debugDraw(e1, e2)
	scale(e2, sf)
	debugDraw(e1, e2)
	rotate(e2, edgeRadians(e1)-edgeRadians(e2)+math.Pi)
	debugDraw(e1, e2)
	translate(e2, e1.Dest().X, e1.Dest().Y)
	debugDraw(e1, e2)
	Splice(e1.Oprev(), e2.Sym())
	Splice(e1.Sym(), e2.Oprev())
	DeleteEdge(e2)
	debugDraw(e1, nil)
}

func main() {
	http.HandleFunc("/", FrontPage)
	http.HandleFunc("/compile", Compile)
	log.Printf("Listening on localhost:1999")
	log.Fatal(http.ListenAndServe("127.0.0.1:1999", nil))
}

func FrontPage(w http.ResponseWriter, req *http.Request) {
	w.Write(frontPageText)
	//	frontPage.Execute(w)
}

//var frontPage = template.Must(template.New("frontPage").Parse(frontPageText)) // HTML template
var frontPageText = []byte(`<!doctype html>
<html>
<head>
<title>Man, I Fold</title>
<style>
body {
	font-size: 18pt;
	font-family: Optima, Calibri, 'DejaVu Sans', sans-serif;
	font-size: 100%;
	line-height: 15pt;
}
#commands { text-align: center }
#errors { height: 20pt; color: #c00; text-align: center }
</style>
<script>
function keyHandler(event) {
	var e = window.event || event;
	if (e.keyCode == 66) { // b
                compile("b");
		e.preventDefault();
		return false;
	}
	if (e.keyCode == 70) { // f
                compile("f");
		e.preventDefault();
		return false;
	}
	if (e.keyCode == 77) { // m
                compile("m");
		e.preventDefault();
		return false;
	}
	if (e.keyCode == 82) { // r
                compile("r");
		e.preventDefault();
		return false;
	}
	if (e.keyCode == 83) { // s
                compile("s");
		e.preventDefault();
		return false;
	}
	if (e.keyCode == 84) { // t
                compile("t");
		e.preventDefault();
		return false;
	}
	if (e.keyCode == 85) { // u
                compile("u");
		e.preventDefault();
		return false;
	}
	if (e.keyCode == 86) { // v
                compile("v");
		e.preventDefault();
		return false;
	}
	if (e.keyCode == 90) { // z
                compile("z");
		e.preventDefault();
		return false;
	}
	if (51 <= e.keyCode && e.keyCode <= 57) { // 3-9
                compile(String.fromCharCode(e.keyCode));
		e.preventDefault();
		return false;
	}
        return true;
}
var xmlreq;
function compile(prog) {
	var req = new XMLHttpRequest();
	xmlreq = req;
	req.onreadystatechange = compileUpdate;
	req.open("POST", "/compile", true);
	req.setRequestHeader("Content-Type", "text/plain; charset=utf-8");
	req.send(prog);
}
function compileUpdate() {
	var req = xmlreq;
	if(!req || req.readyState != 4) {
		return;
	}
	if(req.status == 200) {
		document.getElementById("output").innerHTML = req.responseText;
		document.getElementById("errors").innerHTML = "";
	} else {
		document.getElementById("errors").innerHTML = req.responseText;
	}
}
</script>
</head>
<body onload='compile("z")' onkeydown="keyHandler(event);">
<div id="commands">3&ndash;9: polygon, f: forward, b: back, r: reverse, s: save, t: tab, u: undo, z: zero, m: maximize toggle</div>
<div id="errors"></div>
<div id="output" align="center"></div>
</body>
</html>
`)

var e0 *Edge         // "current" edge, on perimeter in CCW direction in coordinate system with Y coordinates up
var reversed = false // whether arrow on current edge is draw source->target or target->source
var internal = make(map[*QuadEdge]bool)
var tabEdge = make(map[*QuadEdge]bool)
var maximize = false
var history = new(bytes.Buffer)

func attachAndMove(e1 *Edge) {
	if e0 == nil {
		e0 = e1
		return
	}
	internal[e0.Q] = true
	eNext := forwardSkipTabs(e0)
	attach(e0, e1)
	if *eNext == *e0 {
		eNext = e0.Oprev()
	}
	e0 = eNext
}

func ccwPerimeter(e *Edge) *Edge {
	return e.Rprev()
}

func cwPerimeter(e *Edge) *Edge {
	return e.Rnext()
}

func backward(e *Edge) *Edge {
	if reversed {
		return ccwPerimeter(e)
	} else {
		return cwPerimeter(e)
	}
}

func forward(e *Edge) *Edge {
	if reversed {
		return cwPerimeter(e)
	} else {
		return ccwPerimeter(e)
	}
}

func backwardSkipTabs(e *Edge) *Edge {
	e1 := backward(e)
	for tabEdge[e1.Q] && *e1 != *e {
		e1 = backward(e1)
	}
	return e1
}

func forwardSkipTabs(e *Edge) *Edge {
	e1 := forward(e)
	for tabEdge[e1.Q] && *e1 != *e {
		e1 = forward(e1)
	}
	return e1
}

func command(cmd string) error {
	switch string(cmd) {
	case "3":
		attachAndMove(Ngon(3, documentPolygonSide))
	case "4":
		attachAndMove(Ngon(4, documentPolygonSide))
	case "5":
		attachAndMove(Ngon(5, documentPolygonSide))
	case "6":
		attachAndMove(Ngon(6, documentPolygonSide))
	case "7":
		attachAndMove(Ngon(7, documentPolygonSide))
	case "8":
		attachAndMove(Ngon(8, documentPolygonSide))
	case "9":
		attachAndMove(Ngon(9, documentPolygonSide))
	case "b":
		if e0 == nil {
			return nil
		}
		e0 = backwardSkipTabs(e0)
	case "f":
		if e0 == nil {
			return nil
		}
		e0 = forwardSkipTabs(e0)
	case "m":
		maximize = !maximize
	case "r":
		reversed = !reversed
	case "s":
		file, err := os.Create("hello.svg")
		if err != nil {
			log.Fatal(err)
		}
		out := draw(&options{false, false})
		file.Write(out)
		return nil // don't add "s" to command history
	case "t":
		if e0 == nil {
			return nil
		}
		if !tabEdge[e0.Q] { // e0 can be a tab edge if entire perimeter is tabs; don't attach a tab to a tab
			attachAndMove(tab(e0))
		}
	case "u":
		commands := history.String()
		if len(commands) == 0 {
			return nil // nothing to undo
		}
		// undo last command by replaying all commands...
		commands = commands[:len(commands)-1] // ... except last command ...
		command("z")                          // ... starting from zero state.
		for _, cmd := range commands {
			command(string(cmd))
		}
		return nil // don't add "u" to command history
	case "v":
		if e0 == nil {
			return nil
		}
		if !tabEdge[e0.Q] { // e0 can be a tab edge if entire perimeter is tabs; don't attach a tab to a tab
			attachAndMove(traySide())
		}
	case "z":
		e0 = nil
		internal = make(map[*QuadEdge]bool)
		tabEdge = make(map[*QuadEdge]bool)
		reversed = false
		maximize = false
		history = new(bytes.Buffer)
		return nil // don't add "z" to (now empty) command history
	default:
		return fmt.Errorf("Unknown command") // don't add errors to command history
	}
	fmt.Fprintf(history, "%s", cmd) // NB cmd is a single character
	return nil
}

func Compile(w http.ResponseWriter, req *http.Request) {
	cmd, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}
	err = command(string(cmd))
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}
	out := draw(nil)
	w.Write(out) // ignore err
}

type options struct {
	border bool
	cursor bool
}

var documentUnits = "in"
var documentUnitWidth = 11.0
var documentUnitHeight = 8.5
var documentWidth = 1100.0
var documentHeight = 850.0
var documentMargin = 25.0
var documentPolygonSide = 100.0

// convex hull, assuming e0 is an edge on the perimeter of the polygon in ccw orientation
// TODO: use this to find the best fit on the paper
func convexHull() *Edge {
	if e0 == nil {
		return nil
	}
	// make a copy of the perimeter
	n := 1 // n will be the number of points on the perimeter
	for ePath := ccwPerimeter(e0); *ePath != *e0; ePath = ccwPerimeter(ePath) {
		n++
	}
	pts := make([]*Point2D, n)
	pts[0] = e0.Org()
	i := 1
	for ePath := ccwPerimeter(e0); *ePath != *e0; ePath = ccwPerimeter(ePath) {
		pts[i] = ePath.Org()
		i++
	}
	hull := Polygon(pts)
	// function to compute the ccw angle between an edge and its successor,
	// if > Pi then convex else concave
	angle := func(e *Edge) float64 {
		eAngle := edgeRadians(e.Sym())
		if eAngle < 0 {
			eAngle += 2 * math.Pi
		}
		// eAngle is radians to e.Sym() from positive X axis in range [0, 2*PI)
		n := ccwPerimeter(e) // next
		nAngle := edgeRadians(n)
		if nAngle < 0 {
			nAngle += 2 * math.Pi
		}
		// nAngle is radians to n from positive X axis in range [0, 2*PI)
		a := nAngle - eAngle
		if a < 0 {
			a += 2 * math.Pi
		}
		// a is angle between n and e in range [0, 2*PI)
		return a
	}
	// Look at all angles around the perimeter and delete any that are concave
	e := hull
	for {
		if angle(e) > math.Pi { // convex angle
			e = ccwPerimeter(e)
			if *e == *hull {
				break
			}
		} else {
			// concave angle, so delete the point e.Dest()
			e2 := ccwPerimeter(e)
			e3 := ccwPerimeter(e2)
			//
			// *--e-->x--e2-->*--e3-->*
			//
			// we want to delete x (and e2)
			Splice(e.Sym(), e2)
			Splice(e2.Sym(), e3)
			Splice(e.Sym(), e3)
			e.SetDest(e3.Org())
			//
			// *--e---------->*--e3-->*
			//
			hull = e // in case hull was e2 to start with
		}
	}
	return hull
}

func draw(opt *options) []byte {
	printBorder, printCursor := true, true
	if opt != nil {
		printBorder = opt.border
		printCursor = opt.cursor
	}
	buf := new(bytes.Buffer)
	s := svg.New(buf)
	s.Startunit(documentUnitWidth, documentUnitHeight, documentUnits, fmt.Sprintf("viewBox='0 0 %f %f'", documentWidth, documentHeight))
	if printBorder {
		s.Rect(0, 0, documentWidth, documentHeight, "stroke:black; fill:none")
	}
	if e0 == nil {
		s.End()
		return buf.Bytes()
	}
	// arrowhead
	s.Marker("Triangle", 9, 3, 10, 6, "viewBox='0 0 10 6' markerUnits='strokeWidth' orient='auto' fill='red'")
	s.Path("M 0 0 L 10 3 L 0 6 z")
	s.MarkerEnd()
	small, big := BoundingBox(e0)

	// margin
	s.Gtransform(fmt.Sprintf("translate(%f,%f)", documentMargin, documentMargin))

	scale := 1.0
	width := big.X - small.X
	height := big.Y - small.Y
	scaleX := (documentWidth - 2*documentMargin) / width
	scaleY := (documentHeight - 2*documentMargin) / height
	if scaleX < 1 || scaleY < 1 || maximize { // must scale down to fit or up to maximize
		scale = math.Min(scaleX, scaleY)
	}
	if scale != 1 {
		s.Gtransform(fmt.Sprintf("scale(%f)", scale))
	}

	shift := small.X < 0 || small.Y < 0 || maximize
	if shift {
		dx, dy := -small.X, -small.Y
		s.Gtransform(fmt.Sprintf("translate(%f,%f)", dx, dy))
	}

	// Draw the perimeter as one continuous path, for efficient cutting
	pathbuf := new(bytes.Buffer)
	fmt.Fprintf(pathbuf, "M %f %f %f %f", e0.Org().X, e0.Org().Y, e0.Dest().X, e0.Dest().Y)
	for ePath := ccwPerimeter(e0); *ePath != *e0; ePath = ccwPerimeter(ePath) {
		fmt.Fprintf(pathbuf, "L %f %f", ePath.Dest().X, ePath.Dest().Y)
	}
	s.Path(string(pathbuf.Bytes()), "stroke:#000;stroke-width:1;fill:none")

	if debug {
		// Draw the convex hull
		pathbuf.Reset()
		hull := convexHull()
		fmt.Fprintf(pathbuf, "M %f %f %f %f", hull.Org().X, hull.Org().Y, hull.Dest().X, hull.Dest().Y)
		for ePath := ccwPerimeter(hull); *ePath != *hull; ePath = ccwPerimeter(ePath) {
			fmt.Fprintf(pathbuf, "L %f %f", ePath.Dest().X, ePath.Dest().Y)
		}
		s.Path(string(pathbuf.Bytes()), "stroke:#000;stroke-width:3;fill:none")
	}

	// Draw interior edges and the cursor
	for i, e := range e0.Edges() {
		if i == 0 && printCursor {
			if reversed {
				e = e.Sym()
			}
			s.Line(e.Org().X, e.Org().Y,
				e.Dest().X, e.Dest().Y,
				"marker-end='url(#Triangle)' style='stroke:#f00;stroke-width:2'")
		} else if internal[e.Q] {
			s.Line(e.Org().X, e.Org().Y,
				e.Dest().X, e.Dest().Y,
				"stroke:#000;stroke-width:1;stroke-dasharray:1 4")
		}
	}
	if shift {
		s.Gend()
	}
	if scale != 1 {
		s.Gend()
	}
	s.Gend()
	s.End()
	return buf.Bytes()
}
