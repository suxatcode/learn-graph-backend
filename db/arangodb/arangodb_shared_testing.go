package arangodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suxatcode/learn-graph-poc-backend/db"
)

var TESTONLY_Config = db.Config{
	Host:             "http://localhost:18529",
	NoAuthentication: true,
}

func TESTONLY_initdb() {
	db_iface, err := NewArangoDB(TESTONLY_Config)
	if err != nil {
		panic(err)
	}
	db := db_iface.(*ArangoDB)
	ctx := context.Background()
	exists, _ := db.cli.DatabaseExists(ctx, GRAPH_DB_NAME)
	if exists {
		learngraph, _ := db.cli.Database(ctx, GRAPH_DB_NAME)
		learngraph.Remove(ctx)
	}
}
func TESTONLY_SetupAndCleanup(t *testing.T, db db.DB) error {
	testingCleanupDB := func(db *ArangoDB, t *testing.T) {
		if db.db != nil {
			err := db.db.Remove(context.Background())
			assert.NoError(t, err)
		}
		exists, err := db.cli.DatabaseExists(context.Background(), GRAPH_DB_NAME)
		assert.NoError(t, err)
		if !exists {
			return
		}
		thisdb, err := db.cli.Database(context.Background(), GRAPH_DB_NAME)
		assert.NoError(t, err)
		err = thisdb.Remove(context.Background())
		assert.NoError(t, err)
	}
	t.Cleanup(func() { testingCleanupDB(db.(*ArangoDB), t) })
	err := db.(*ArangoDB).CreateDBWithSchema(context.Background())
	assert.NoError(t, err)
	return err
}