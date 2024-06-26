package controller

import (
	"context"
	"testing"

	"github.com/quartercastle/vector"
	"github.com/stretchr/testify/assert"
	"github.com/suxatcode/learn-graph-poc-backend/graph/model"
	"github.com/suxatcode/learn-graph-poc-backend/layout"
)

func TestForceSimulationLayouter_GetNodePositions_simple(t *testing.T) {
	l := NewForceSimulationLayouter()
	l.simulationState.lnodes = []*layout.Node{
		{Name: "1", Pos: vector.Vector{1, 2, 3}}, {Name: "2", Pos: vector.Vector{3, 4, 5}},
	}
	l.simulationState.modelToLayoutNodeLookup = map[string]int{
		"1": 0,
		"2": 1,
	}
	g := &model.Graph{
		Nodes: []*model.Node{{ID: "1"}, {ID: "2"}},
	}
	close(l.waitForInitialLayout) // assume initial layout is there
	l.GetNodePositions(context.Background(), g)
	assert := assert.New(t)
	assert.Equal([]*model.Node{
		{ID: "1", Position: &model.Vector{X: 1, Y: 2, Z: 3}}, {ID: "2", Position: &model.Vector{X: 3, Y: 4, Z: 5}},
	}, g.Nodes)
}

func TestForceSimulationLayouter_GetNodePositions_notOrdered(t *testing.T) {
	l := NewForceSimulationLayouter()
	l.simulationState.lnodes = []*layout.Node{
		{Name: "2", Pos: vector.Vector{3, 4, 5}}, {Name: "1", Pos: vector.Vector{1, 2, 3}},
	}
	l.simulationState.modelToLayoutNodeLookup = map[string]int{
		"2": 0,
		"1": 1,
	}
	g := &model.Graph{
		Nodes: []*model.Node{{ID: "2"}, {ID: "1"}},
	}
	close(l.waitForInitialLayout) // assume initial layout is there
	l.GetNodePositions(context.Background(), g)
	assert := assert.New(t)
	assert.Equal([]*model.Node{
		{ID: "2", Position: &model.Vector{X: 3, Y: 4, Z: 5}}, {ID: "1", Position: &model.Vector{X: 1, Y: 2, Z: 3}},
	}, g.Nodes)
}

func TestForceSimulationLayouter_GetNodePositions_missingNodes(t *testing.T) {
	l := NewForceSimulationLayouter()
	l.simulationState.lnodes = []*layout.Node{
		{Name: "1", Pos: vector.Vector{1, 2, 3}}, {Name: "2", Pos: vector.Vector{3, 4, 5}},
	}
	l.simulationState.modelToLayoutNodeLookup = map[string]int{
		"1": 0,
		"2": 1,
	}
	l.simulationState.modelToLayoutEdgeLookup = map[string]int{}
	g := &model.Graph{
		Nodes: []*model.Node{{ID: "1"}, {ID: "2"}, {ID: "3"}},
	}
	close(l.waitForInitialLayout) // assume initial layout is there
	l.GetNodePositions(context.Background(), g)
	assert := assert.New(t)
	for i, expected := range []*model.Node{
		{ID: "1", Position: &model.Vector{X: 1, Y: 2, Z: 3}},
		{ID: "2", Position: &model.Vector{X: 3, Y: 4, Z: 5}},
		{ID: "3", Position: &model.Vector{X: 58.737482042038394, Y: 133.03387516414173, Z: 0}}, // XXX(skep): not ready for 3D
	} {
		assert.Equal(expected.ID, g.Nodes[i].ID)
		assert.True(layout.IsClose(expected.Position.X, g.Nodes[i].Position.X), "expected '%v', but got '%v'", expected.Position, g.Nodes[i].Position)
		assert.True(layout.IsClose(expected.Position.Y, g.Nodes[i].Position.Y), "expected '%v', but got '%v'", expected.Position, g.Nodes[i].Position)
		assert.True(layout.IsClose(expected.Position.Z, g.Nodes[i].Position.Z), "expected '%v', but got '%v'", expected.Position, g.Nodes[i].Position)
	}
}

// snapshot test that force simulation is executed
func TestForceSimulationLayouter_Reload(t *testing.T) {
	l := NewForceSimulationLayouter()
	g := &model.Graph{
		Nodes: []*model.Node{{ID: "2", Description: "B"}, {ID: "1", Description: "A"}},
		Edges: []*model.Edge{{ID: "55", From: "1", To: "2", Weight: 5.0}},
	}
	l.Reload(context.Background(), g)
	assert := assert.New(t)
	for i, node := range []*layout.Node{
		{Name: "B", Pos: vector.Vector{-0.11307979551978042, -4.8444672122163555}},
		{Name: "A", Pos: vector.Vector{0.11307979551978042, 4.8444672122163555}},
	} {
		assert.Equal(node.Name, l.simulationState.lnodes[i].Name)
		assert.True(layout.IsCloseVec(node.Pos, l.simulationState.lnodes[i].Pos, 0, 0.02), "expected '%v' to be close to '%v' (relative tolerance 0.02)", node.Pos, l.simulationState.lnodes[i].Pos)
	}
	assert.Equal(map[string]int{"2": 0, "1": 1}, l.simulationState.modelToLayoutNodeLookup)
	assert.Equal(map[string]int{"55": 0}, l.simulationState.modelToLayoutEdgeLookup)
}
