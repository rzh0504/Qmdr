package main

import (
	"math"
	"runtime"

	"github.com/egoist/mygo/ui"
)

// cornerExtent is how far the continuous corner's curve runs along an
// edge, in radii, as Core Animation draws it.
const cornerExtent = 1.528665

// smoothCornerCurve is a unit-radius continuous corner from its top edge
// to its left edge: points (u, v) inside those edges, from (extent, 0) to
// (0, extent).
var smoothCornerCurve = buildSmoothCorner(24)

func buildSmoothCorner(steps int) []ui.Point {
	extent := float32(cornerExtent)
	points := make([]ui.Point, 0, steps+1)
	for i := 0; i <= steps; i++ {
		v := extent * float32(i) / float32(steps)
		lo, hi := float32(0), extent
		for range 24 {
			mid := (lo + hi) / 2
			if cornerDistance(mid, v) > 0 {
				lo = mid
			} else {
				hi = mid
			}
		}
		points = append(points, ui.Point{X: (lo + hi) / 2, Y: v})
	}
	return points
}

// cornerDistance is the continuous corner's value at (u, v), in a radius
// of one: about the signed distance to its edge near it, positive outside.
func cornerDistance(u, v float32) float32 {
	extent := float32(cornerExtent)
	ax := max(0, 1-u/extent)
	ay := max(0, 1-v/extent)
	l := float32(math.Sqrt(float64(ax*ax + ay*ay)))
	hi, lo := max(ax, ay), min(ax, ay)
	rho := float32(0)
	if hi > 0 {
		rho = min(lo/hi, 1)
	}
	p := (((-0.926054*rho+3.15601)*rho-3.64122)*rho+1.26803)*rho + 0.268531
	f := l + 1 - 1/(1-rho*rho*min(l, 1)*p)
	return min(max(extent-u, extent-v), 0) + extent*(f-1)
}

// smoothRectPath returns the continuous rounded rectangle of radius, as
// MyGo draws on macOS, for renderers that round corners circularly.
func smoothRectPath(r ui.Rect, radius float32) ui.Path {
	rad := min(radius, min(r.W, r.H)/2)
	rad = min(rad, min(r.W, r.H)/(2*cornerExtent))
	var p ui.Path
	if rad <= 0 || r.W <= 0 || r.H <= 0 {
		p.MoveTo(r.X, r.Y).LineTo(r.X+r.W, r.Y).
			LineTo(r.X+r.W, r.Y+r.H).LineTo(r.X, r.Y+r.H).Close()
		return p
	}
	extent := rad * cornerExtent
	points := smoothCornerCurve

	p.MoveTo(r.X+extent, r.Y)
	p.LineTo(r.X+r.W-extent, r.Y)
	for _, pt := range points {
		p.LineTo(r.X+r.W-pt.X*rad, r.Y+pt.Y*rad)
	}
	p.LineTo(r.X+r.W, r.Y+r.H-extent)
	for i := len(points) - 1; i >= 0; i-- {
		pt := points[i]
		p.LineTo(r.X+r.W-pt.X*rad, r.Y+r.H-pt.Y*rad)
	}
	p.LineTo(r.X+extent, r.Y+r.H)
	for _, pt := range points {
		p.LineTo(r.X+pt.X*rad, r.Y+r.H-pt.Y*rad)
	}
	p.LineTo(r.X, r.Y+extent)
	for i := len(points) - 1; i >= 0; i-- {
		pt := points[i]
		p.LineTo(r.X+pt.X*rad, r.Y+pt.Y*rad)
	}
	p.Close()
	return p
}

// smoothFlat fills a flat surface, with no shadow, in a continuous
// rounded rectangle on platforms whose renderer rounds corners
// circularly.
func smoothFlat(e ui.Element, col colors, radius float32, fill ui.Color) ui.Element {
	e.Radius(radius)
	if runtime.GOOS == "darwin" {
		e.Background(fill)
		return e
	}
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		path := smoothRectPath(r, radius)
		p.FillPath(&path, fill)
	})
	return e
}

// smoothGradientTile fills a tile with its base and glow gradients in a
// continuous rounded rectangle, with its shadow and dark edge.
func smoothGradientTile(e ui.Element, col colors, radius float32, base, glow ui.LinearGradient) ui.Element {
	e.Radius(radius)
	e = lift(e, col)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		path := smoothRectPath(r, radius)
		p.FillPathGradient(&path, base)
		p.FillPathGradient(&path, glow)
		if col.dark {
			p.StrokePath(&path, 1, col.edge)
		}
	})
	return e
}

// smoothPanel fills e's surface with a continuous rounded rectangle on
// platforms whose renderer rounds corners circularly; on macOS MyGo draws
// it continuously already, and e keeps its background, border and clip.
func smoothPanel(e ui.Element, col colors, radius float32, fill ui.Color, clip bool) ui.Element {
	e.Radius(radius)
	if runtime.GOOS == "darwin" {
		e.Background(fill)
		e = raised(e, col)
		if clip {
			e.Clip()
		}
		return e
	}
	e = lift(e, col)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		path := smoothRectPath(r, radius)
		p.FillPath(&path, fill)
		if col.dark {
			p.StrokePath(&path, 1, col.edge)
		}
	})
	return e
}
