package controller

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/suxatcode/learn-graph-poc-backend/db"
	"github.com/suxatcode/learn-graph-poc-backend/graph/model"
	"github.com/suxatcode/learn-graph-poc-backend/middleware"
)

const (
	AuthNeededForGraphDataChangeMsg = `only logged in user may create graph data`
)

var (
	AuthNeededForGraphDataChangeErr    = errors.New(AuthNeededForGraphDataChangeMsg)
	AuthNeededForGraphDataChangeResult = &model.CreateEntityResult{Status: &model.Status{Message: AuthNeededForGraphDataChangeMsg}}
)

type Controller struct {
	db db.DB
}

func NewController(newdb db.DB) *Controller {
	return &Controller{db: newdb}
}

func (c *Controller) CreateNode(ctx context.Context, description model.Text) (*model.CreateEntityResult, error) {
	authenticated, user, err := c.db.IsUserAuthenticated(ctx)
	if err != nil || !authenticated || user == nil {
		if err != nil {
			log.Ctx(ctx).Error().Msgf("%v", err)
			return nil, err
		}
		log.Ctx(ctx).Error().Msgf("user '%s' (token '%s') not authenticated", middleware.CtxGetUserID(ctx), middleware.CtxGetAuthentication(ctx))
		return AuthNeededForGraphDataChangeResult, AuthNeededForGraphDataChangeErr
	}
	id, err := c.db.CreateNode(ctx, *user, &description)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("%v", err)
		return nil, err
	}
	res := &model.CreateEntityResult{ID: id}
	log.Ctx(ctx).Debug().Msgf("CreateNode() -> %v", res)
	return res, nil
}

func (c *Controller) CreateEdge(ctx context.Context, from string, to string, weight float64) (*model.CreateEntityResult, error) {
	authenticated, user, err := c.db.IsUserAuthenticated(ctx)
	if err != nil || !authenticated || user == nil {
		if err != nil {
			log.Ctx(ctx).Error().Msgf("%v", err)
			return nil, err
		}
		log.Ctx(ctx).Error().Msgf("user '%s' (token '%s') not authenticated", middleware.CtxGetUserID(ctx), middleware.CtxGetAuthentication(ctx))
		return AuthNeededForGraphDataChangeResult, AuthNeededForGraphDataChangeErr
	}
	from, to = db.AddNodePrefix(from), db.AddNodePrefix(to)
	ID, err := c.db.CreateEdge(ctx, *user, from, to, weight)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("%v", err)
		return nil, err
	}
	res := &model.CreateEntityResult{ID: ID}
	log.Ctx(ctx).Debug().Msgf("CreateEdge() -> %v", res)
	return res, nil
}

func (c *Controller) EditNode(ctx context.Context, id string, description model.Text) (*model.Status, error) {
	err := c.db.EditNode(ctx, id, &description)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("%v", err)
		return nil, err
	}
	log.Ctx(ctx).Debug().Msgf("EditNode() -> %v", nil)
	return nil, nil
}

func (c *Controller) SubmitVote(ctx context.Context, id string, value float64) (*model.Status, error) {
	err := c.db.SetEdgeWeight(ctx, id, value)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("%v", err)
		return nil, err
	}
	log.Ctx(ctx).Debug().Msgf("SubmitVote() -> %v", nil)
	return nil, nil
}

func (c *Controller) Graph(ctx context.Context) (*model.Graph, error) {
	g, err := c.db.Graph(ctx)
	if err != nil || g == nil {
		log.Ctx(ctx).Error().Msgf("%v | graph=%v", err, g)
	} else if g != nil {
		log.Ctx(ctx).Debug().Msgf("returns %d nodes and %d edges", len(g.Nodes), len(g.Edges))
	}
	return g, err
}