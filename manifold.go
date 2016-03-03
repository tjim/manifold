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

func tab() *Edge {
	pts := []*Point2D{{0.0, 0.0}, {40.0, 0.0}, {30.0, 10.0}, {10.0, 10.0}}
	p := Polygon(pts)
	for _, e := range p.Edges() {
		if *e == *p {
			continue
		}
		tabEdge[e.Q] = true
	}
	return p
}

func tab2() *Edge {
	pts := []*Point2D{{0.0, 0.0}, {10.0, 0.0}, {0.0, 20.0}, {-10.0, 0.0}}
	p := Polygon(pts)
	for _, e := range p.Edges() {
		if *e == *p {
			continue
		}
		tabEdge[e.Q] = true
	}
	return p
}

func splitBack(e *Edge) *Edge {
	// split the edge e into two edges:
	// A----e---->B becomes A----e2---->A'----e---->B
	// New vertex A' has the same geometric coordinates as A
	// Return the new edge e2
	org := e.Org()
	prev := e.Oprev()
	Splice(e, prev)
	e1 := MakeEdge()
	Splice(e1, prev)
	Splice(e1.Sym(), e)
	e1.SetOrg(org)
	e1.SetDest(org)
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
}
pre, textarea {
	font-family: Optima, Calibri, 'DejaVu Sans', sans-serif;
	font-size: 100%;
	line-height: 15pt;
}
#edit, #output, #errors { width: 100%; text-align: left; }
#output { height: 100%; }
#errors { color: #c00; }
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
	prog = prog || document.getElementById("edit").value;
	document.getElementById("edit").value = "";
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
		document.getElementById("output").innerHTML = "";
	}
}
</script>
</head>
<body onload='compile("z")'>
3-9: polygon, f: forward, b: back, r: reverse, s: save, t: tab, z: zero, m: maximize toggle<br />
<input autofocus="true" id="edit" onkeydown="keyHandler(event);"></input>
<div id="output"></div>
<div id="errors"></div>
</body>
</html>
`)

var e0 *Edge
var outright = true
var internal = make(map[*QuadEdge]bool)
var tabEdge = make(map[*QuadEdge]bool)
var maximize = false

func attachAndMove(e1 *Edge) {
	if e0 == nil {
		e0 = e1
	} else {
		internal[e0.Q] = true
		eNext := forwardSkipTabs(e0)
		if outright {
			attach(e0, e1)
			if *eNext == *e0 {
				eNext = e0.Oprev()
			}
		} else {
			attach(e0.Sym(), e1)
			if *eNext == *e0 {
				eNext = e0.Onext()
			}
		}
		e0 = eNext
	}
}

func backward(e *Edge) *Edge {
	if outright {
		return e.Oprev().Sym()
	} else {
		return e.Onext().Sym()
	}
}

func backwardSkipTabs(e *Edge) *Edge {
	e1 := backward(e)
	for tabEdge[e1.Q] && *e1 != *e {
		e1 = backward(e1)
	}
	return e1
}

func forward(e *Edge) *Edge {
	if outright {
		return e.Sym().Onext()
	} else {
		return e.Sym().Oprev()
	}
}

func forwardSkipTabs(e *Edge) *Edge {
	e1 := forward(e)
	for tabEdge[e1.Q] && *e1 != *e {
		e1 = forward(e1)
	}
	return e1
}

func Compile(w http.ResponseWriter, req *http.Request) {
	cmd, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	switch string(cmd) {
	case "3":
		attachAndMove(Ngon(3, 40))
	case "4":
		attachAndMove(Ngon(4, 40))
	case "5":
		attachAndMove(Ngon(5, 40))
	case "6":
		attachAndMove(Ngon(6, 40))
	case "7":
		attachAndMove(Ngon(7, 40))
	case "8":
		attachAndMove(Ngon(8, 40))
	case "9":
		attachAndMove(Ngon(9, 40))
	case "b":
		e0 = backwardSkipTabs(e0)
	case "f":
		e0 = forwardSkipTabs(e0)
	case "m":
		maximize = !maximize
	case "r":
		e0 = e0.Sym()
		outright = !outright
	case "s":
		file, err := os.Create("hello.svg")
		if err != nil {
			log.Fatal(err)
		}
		out := draw(&options{false, false})
		file.Write(out)
	case "t":
		if !tabEdge[e0.Q] { // e0 can be a tab edge if entire perimeter is tabs; don't attach a tab to a tab
			attachAndMove(tab())
		}
	case "v":
		if !tabEdge[e0.Q] { // e0 can be a tab edge if entire perimeter is tabs; don't attach a tab to a tab
			e1 := splitBack(e0)
			tabEdge[e1.Q] = true
			eLen := edgeLength(e0)
			eRad := edgeRadians(e0)
			dy, dx := math.Sincos(eRad)
			dx, dy = dx*(eLen-10.0), dy*(eLen-10.0)
			org := e0.Org()
			mid := &Point2D{org.X + dx, org.Y + dy}
			e1.SetDest(mid)
			e0.SetOrg(mid)
			attachAndMove(tab2())
		}
	case "z":
		e0 = nil
		internal = make(map[*QuadEdge]bool)
		tabEdge = make(map[*QuadEdge]bool)
		outright = true
		maximize = false
	default:
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
	s.Marker("Triangle", 0, 5, 20, 10, "viewBox='0 0 10 10' markerUnits='strokeWidth' orient='auto'")
	s.Path("M 0 0 L 10 5 L 0 10 z")
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

	for i, e := range e0.Edges() {
		if i == 0 && printCursor {
			s.Line(e.Org().X, e.Org().Y,
				e.Dest().X, e.Dest().Y,
				"marker-end='url(#Triangle)' style='stroke:#f00;stroke-width:1'")
		} else if internal[e.Q] {
			s.Line(e.Org().X, e.Org().Y,
				e.Dest().X, e.Dest().Y,
				"stroke:#000;stroke-width:1;stroke-dasharray:4")
		} else {
			s.Line(e.Org().X, e.Org().Y,
				e.Dest().X, e.Dest().Y,
				"stroke:#000;stroke-width:1")
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
