# Man, I Fold

Manifold is a program for creating paper fold-up models.  For example,
here is a model that can be folded up into a dodecahedron:

![Dodecahedron](dodecahedron.png)

To run manifold, first install and configure [go](http://golang.org/),
then

    go run manifold.go

Manifold is a keyboard-driven web application available at
`localhost:1999`.  Use the keys `3`-`9` to add a regular polygon to
your model.  Once a polygon has been inserted, an edge on the
perimeter of the model will be highlighted as a red arrow; this is
called the *cursor*.  The cursor indicates where the next polygon will
be added to the model.  You can move the cursor forward and backward
along the perimeter of the model using the `f` and `b` keys.  Reverse
the direction of the cursor with `r`.  Add a tab (for gluing the edges
of the model together) with `t`.  You can start fresh by hitting `z`.

Once you are satisfied with the model, save it to the file `hello.svg`
by entering `s`.  (There is no way to use another filename.)  You can
open this file with your browser, print it, cut it out, fold it, and
glue it together.  Or you can use a paper cutting machine to do all of
the cutting.  (I have tested this with a Silhouette Cameo.)


