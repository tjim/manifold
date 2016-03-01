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
var frontPageText =  []byte(`<!doctype html>
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
#errors { color: #c00; }
</style>
<script>
function keyHandler(event) {
    var e = window.event || event;
    if (e.keyCode == 13) { // enter
            compile(e.target);
            e.preventDefault();
            return false;
    }
    return true;
}
var xmlreq;
function compile() {
	var prog = document.getElementById("edit").value;
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
<body>
<input autofocus="true" id="edit" onkeydown="keyHandler(event);"></input>
<div id="output"></div>
<div id="errors"></div>
</body>
</html>
`)

var e0 *Edge

func Compile(w http.ResponseWriter, req *http.Request) {
	log.Printf("Compile\n")
	cmd, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	log.Printf("Your command: %s\n", cmd)
	switch string(cmd) {
	case "tri":
		e1 := Ngon(3, 40)
		if e0 == nil {
			e0 = e1
		} else {
			attach(e0, e1)
		}
	case "hex":
		e1 := Ngon(6, 40)
		if e0 == nil {
			e0 = e1
		} else {
			attach(e0, e1)
		}
	case "l":
		e0 = e0.Lnext()
	case "r":
		e0 = e0.Rprev()
	case "tl":
		e0 = e0.Onext()
	case "tr":
		e0 = e0.Oprev()
	case "ta":
		e0 = e0.Sym()
	case "box":
		fallthrough
	default:
		e1 := Ngon(4, 40)
		if e0 == nil {
			e0 = e1
		} else {
			attach(e0, e1)
		}
	}
	out := draw()
	w.Write(out) // ignore err
}

func draw() []byte {
	log.Println("draw()")
	buf := new(bytes.Buffer)
	s := svg.New(buf)
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
	s.End()
	log.Println("draw() done")
	return buf.Bytes()
}
