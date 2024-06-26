package layout

import (
	"context"
	"math/rand"
	"testing"

	"github.com/quartercastle/vector"
	"github.com/stretchr/testify/assert"
)

func TestForceSimulation_ComputeLayoutShouldBreakOnCtxDone(t *testing.T) {
	fs := NewForceSimulation(DefaultForceSimulationConfig)
	cancelled_ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p0, p1 := vector.Vector{1, 1}, vector.Vector{2, 2}
	nodes, _ := fs.ComputeLayout(cancelled_ctx, []*Node{{Pos: p0}, {Pos: p1}}, []*Edge{})
	assert := assert.New(t)
	assert.Equal(p0, nodes[0].Pos)
	assert.Equal(p1, nodes[1].Pos)
}

func TestForceSimulation_ComputeLayout(t *testing.T) {
	for _, test := range []struct {
		Name       string
		Config     ForceSimulationConfig
		Nodes      []*Node
		Edges      []*Edge
		Assertions func(t *testing.T, nodes []*Node)
	}{
		{
			Name:  "push 2 nodes apart (even though connected!)",
			Nodes: []*Node{{Name: "A", Pos: vector.Vector{9, 9}}, {Name: "B", Pos: vector.Vector{10, 10}}},
			Edges: []*Edge{{Source: 0, Target: 1}},
			Assertions: func(t *testing.T, nodes []*Node) {
				assert := assert.New(t)
				assert.Less(nodes[0].Pos.X(), 8.0)
				assert.Less(nodes[0].Pos.Y(), 8.0)
				assert.Greater(nodes[1].Pos.X(), 10.0)
				assert.Greater(nodes[1].Pos.Y(), 10.0)
			},
			Config: ForceSimulationConfig{RandomFloat: func() float64 { return 1.0 }},
		},
		{
			Name:  "pull 2 nodes together by edge",
			Nodes: []*Node{{Name: "A", Pos: vector.Vector{1, 1}}, {Name: "B", Pos: vector.Vector{200, 200}}},
			Edges: []*Edge{{Source: 0, Target: 1}},
			Assertions: func(t *testing.T, nodes []*Node) {
				assert := assert.New(t)
				// expect, that the equilibrium settles somewhere around P=(100,100)
				// n1, n2 ∈ (90, 100)
				assert.Greater(nodes[0].Pos.X(), 90.0)
				assert.Greater(nodes[0].Pos.Y(), 90.0)
				assert.Less(nodes[0].Pos.X(), 110.0)
				assert.Less(nodes[0].Pos.Y(), 110.0)
				assert.Greater(nodes[1].Pos.X(), 90.0)
				assert.Greater(nodes[1].Pos.Y(), 90.0)
				assert.Less(nodes[1].Pos.X(), 110.0)
				assert.Less(nodes[1].Pos.Y(), 110.0)
			},
			Config: ForceSimulationConfig{RandomFloat: func() float64 { return 1.0 }},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			fs := NewForceSimulation(test.Config)
			nodes, stats := fs.ComputeLayout(context.Background(), test.Nodes, test.Edges)
			test.Assertions(t, nodes)
			assert.NotZero(t, stats.TotalTime.Nanoseconds())
			assert.Equal(t, 223, stats.Iterations)
		})
	}
}

func TestForceSimulation_calculateAttractionForce(t *testing.T) {
	fs := NewForceSimulation(DefaultForceSimulationConfig)
	graph := NewGraph([]*Node{{Pos: vector.Vector{1, 1}}, {Pos: vector.Vector{4, 5}}}, []*Edge{}, fs)
	from, to := graph.Nodes[0], graph.Nodes[1]
	force := fs.calculateAttractionForce(from, to, 1.0)
	assert := assert.New(t)
	assert.Equal(vector.Vector{-1.7999999999999998, -2.4000000000000004}, force)

	graph = NewGraph([]*Node{{Pos: vector.Vector{10, 10}}, {Pos: vector.Vector{40, 50}}}, []*Edge{}, fs)
	from, to = graph.Nodes[0], graph.Nodes[1]
	force = fs.calculateAttractionForce(from, to, 1.0)
	assert.Equal(vector.Vector{-28.799999999999997, -38.400000000000006}, force)
}

func TestForceSimulation_calculateRepulsionForce(t *testing.T) {
	// conf := ForceSimulationConfig{RepulsionMultiplier: 10.0}
	fs := NewForceSimulation(DefaultForceSimulationConfig)
	graph := NewGraph([]*Node{{Pos: vector.Vector{1, 1}}, {Pos: vector.Vector{4, 5}}}, []*Edge{}, fs)
	from, to := graph.Nodes[0], graph.Nodes[1]
	force, tmp := vector.Vector{0, 0}, vector.Vector{0, 0}
	fs.calculateRepulsionForce(&force, &tmp, from, to)
	assert := assert.New(t)
	assert.Equal(vector.Vector{-1.2, -1.6}, force)

	graph = NewGraph([]*Node{{Pos: vector.Vector{10, 10}}, {Pos: vector.Vector{40, 50}}}, []*Edge{}, fs)
	from, to = graph.Nodes[0], graph.Nodes[1]
	force, tmp = vector.Vector{0, 0}, vector.Vector{0, 0}
	fs.calculateRepulsionForce(&force, &tmp, from, to)
	assert.Equal(vector.Vector{-0.12, -0.16000000000000003}, force)
}

func BenchmarkForceSimulation_ComputeLayout(b *testing.B) {
	for n := 10; n < b.N; n += 10 {
		fs := NewForceSimulation(DefaultForceSimulationConfig)
		nodes := []*Node{}
		edges := []*Edge{}
		for i := 0; i < n; i++ {
			nodes = append(nodes, &Node{})
		}
		for i := 0; i < n; i++ {
			edge := Edge{Source: rand.Intn(n), Target: rand.Intn(n)}
			if edge.Source == edge.Target {
				if edge.Target == n {
					edge.Target = edge.Source - 1
				} else {
					edge.Target = edge.Source + 1
				}
			}
			edges = append(edges, &edge)
		}
		b.StartTimer()
		fs.ComputeLayout(context.Background(), nodes, edges)
		b.StopTimer()
	}
}
