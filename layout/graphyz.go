// adapted from https://github.com/jwhandley/graphyz/blob/main/g.go
package layout

import (
	"math"

	"github.com/quartercastle/vector"
)

type Body interface {
	size() float64
	position() vector.Vector
}

type Graph struct {
	Nodes           []*Node
	Edges           []*Edge
	forceSimulation *ForceSimulation
}

type Node struct {
	Name     string
	Label    string
	degree   float64
	isPinned bool
	radius   float64
	Pos      vector.Vector
	vel      vector.Vector
	acc      vector.Vector
}

type Edge struct {
	Source int
	Target int
	Value  float64
}

func randomVectorInside(rect Rect, rndSource func() float64) vector.Vector {
	return vector.Vector{
		rect.X + rndSource()*rect.Width,
		rect.Y + rndSource()*rect.Height,
	}
}

func NewGraph(nodes []*Node, edges []*Edge, forceSimulation *ForceSimulation) *Graph {
	graph := Graph{
		Nodes:           nodes,
		Edges:           edges,
		forceSimulation: forceSimulation,
	}
	for _, node := range graph.Nodes {
		if node.Pos.Magnitude() == 0 {
			node.Pos = randomVectorInside(forceSimulation.conf.Rect, forceSimulation.conf.RandomFloat)
		}
		if node.radius == 0 {
			node.radius = forceSimulation.conf.DefaultNodeRadius
		}
		if len(node.acc) == 0 {
			node.acc = vector.Vector{0, 0}
		}
		if len(node.vel) == 0 {
			node.vel = vector.Vector{0, 0}
		}
	}
	for _, edge := range graph.Edges {
		if edge.Value == 0.0 {
			edge.Value = 1.0
		}
		graph.Nodes[edge.Source].degree += edge.Value
		graph.Nodes[edge.Target].degree += edge.Value
	}
	return &graph
}

// XXX: unused
func (g *Graph) resetPosition() {
	var initialRadius float64 = 10.0
	initialAngle := float64(math.Pi) * (3 - math.Sqrt(5))
	for i, node := range g.Nodes {
		radius := initialRadius * float64(math.Sqrt(0.5+float64(i)))
		angle := float64(i) * initialAngle

		node.Pos = vector.Vector{
			radius*float64(math.Cos(angle)) + float64(config.ScreenWidth)/2,
			radius*float64(math.Sin(angle)) + float64(config.ScreenHeight)/2,
		}
	}
}

func (g *Graph) ApplyForce(deltaTime float64, qt *QuadTree) {
	g.resetAcceleration()
	if config.Gravity {
		g.gravityToCenterForce()
	}

	g.attractionByEdgesForce()

	if config.BarnesHut {
		g.repulsionBarnesHut(qt)
	} else {
		g.repulsionNaive()
	}

	g.updatePositions(deltaTime)
}

func clamp(in, lo, hi float64) float64 {
	if math.IsNaN(in) {
		return in
	}
	if in > hi {
		return hi
	} else if in < lo {
		return lo
	}
	return in
}

func VectorClampValue(v vector.Vector, min, max float64) vector.Vector {
	return vector.Vector{
		clamp(v.X(), min, max),
		clamp(v.Y(), min, max),
	}
}
func VectorClampVector(v, min, max vector.Vector) vector.Vector {
	return vector.Vector{
		clamp(v.X(), min.X(), max.X()),
		clamp(v.Y(), min.Y(), max.Y()),
	}
}

func (g *Graph) updatePositions(deltaTime float64) {
	outOfBoundsFactor := g.forceSimulation.conf.ScreenMultiplierToClampPosition
	boundsMin := vector.Vector{
		-outOfBoundsFactor * float64(config.ScreenWidth), -outOfBoundsFactor * float64(config.ScreenHeight),
	}
	boundsMax := vector.Vector{
		outOfBoundsFactor * float64(config.ScreenWidth), outOfBoundsFactor * float64(config.ScreenHeight),
	}
	for _, node := range g.Nodes {
		if !node.isPinned {
			vector.In(node.vel).Add(node.acc)
			vector.In(node.vel).Scale(1 - config.VelocityDecay)
			node.vel = VectorClampValue(node.vel, -100, 100)
			vector.In(node.Pos).Add(node.vel.Scale(deltaTime))
			node.Pos = VectorClampVector(node.Pos, boundsMin, boundsMax)
		}
	}
}

func (g *Graph) resetAcceleration() {
	for _, node := range g.Nodes {
		node.acc = vector.Vector{0, 0}
	}
}

func (g *Graph) gravityToCenterForce() {
	center := vector.Vector{
		float64(config.ScreenWidth) / 2,
		float64(config.ScreenHeight) / 2,
	}
	for _, node := range g.Nodes {
		delta := center.Sub(node.Pos)
		force := delta.Scale(config.GravityStrength * node.size() * g.forceSimulation.temperature)
		node.acc = node.acc.Add(force)
	}
}

func (g *Graph) attractionByEdgesForce() {
	for _, edge := range g.Edges {
		from := g.Nodes[edge.Source]
		to := g.Nodes[edge.Target]
		force := g.forceSimulation.calculateAttractionForce(from, to, edge.Value)
		from.acc = from.acc.Sub(force)
		to.acc = to.acc.Add(force)

	}
}

func (g *Graph) repulsionBarnesHut(qt *QuadTree) {
	qt.Clear()
	for _, node := range g.Nodes {
		qt.Insert(node)
	}
	qt.CalculateMasses()
	for _, node := range g.Nodes {
		force := qt.CalculateForce(node, config.Theta)
		node.acc = node.acc.Add(force)
	}
}

func (g *Graph) repulsionNaive() {
	for i, node := range g.Nodes {
		for j, other := range g.Nodes {
			if i == j {
				continue
			}
			force := g.forceSimulation.calculateRepulsionForce(node, other)
			node.acc = node.acc.Add(force)
		}

	}
}

func (node *Node) size() float64 {
	return node.degree
}

func (node *Node) position() vector.Vector {
	return node.Pos
}
