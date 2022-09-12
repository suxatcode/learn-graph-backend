package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suxatcode/learn-graph-poc-backend/graph/model"
)

func TestModelFromDB(t *testing.T) {
	for _, test := range []struct {
		Name string
		Exp  *model.Graph
		InpV []Node
		InpE []Edge
	}{
		{
			Name: "single vertex",
			InpV: []Node{{Document: Document{Key: "abc"}}},
			Exp: &model.Graph{
				Nodes: []*model.Node{
					{ID: "abc"},
				},
			},
		},
		{
			Name: "multiple vertices",
			InpV: []Node{
				{Document: Document{Key: "abc"}},
				{Document: Document{Key: "def"}},
			},
			Exp: &model.Graph{
				Nodes: []*model.Node{
					{ID: "abc"},
					{ID: "def"},
				},
			},
		},
		{
			Name: "2 vertices 1 edge",
			InpV: []Node{
				{Document: Document{Key: "a"}},
				{Document: Document{Key: "b"}},
			},
			InpE: []Edge{
				{Document: Document{Key: "?"}, From: "a", To: "b"},
			},
			Exp: &model.Graph{
				Nodes: []*model.Node{
					{ID: "a"},
					{ID: "b"},
				},
				Edges: []*model.Edge{
					{ID: "?", From: "a", To: "b"},
				},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Exp, ModelFromDB(test.InpV, test.InpE))
		})
	}
}
