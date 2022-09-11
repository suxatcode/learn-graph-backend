//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/arangodb/go-driver"
	"github.com/stretchr/testify/assert"
	"github.com/suxatcode/learn-graph-poc-backend/graph/model"
)

var testConfig = Config{
	Host:             "http://localhost:18529",
	NoAuthentication: true,
}

func TestNewArangoDB(t *testing.T) {
	_, err := NewArangoDB(testConfig)
	assert.NoError(t, err, "expected connection succeeds")
}

func SetupDB(db *ArangoDB, t *testing.T) {
	db.CreateDBWithSchema(context.Background())
}

func CleanupDB(db *ArangoDB, t *testing.T) {
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

func dbTestSetupCleanup(t *testing.T) (DB, *ArangoDB, error) {
	db, err := NewArangoDB(testConfig)
	assert.NoError(t, err)
	t.Cleanup(func() { CleanupDB(db.(*ArangoDB), t) })
	SetupDB(db.(*ArangoDB), t)
	return db, db.(*ArangoDB), err
}

func TestArangoDB_CreateDBWithSchema(t *testing.T) {
	dbTestSetupCleanup(t)
}

func TestArangoDB_Graph(t *testing.T) {
	db, d, err := dbTestSetupCleanup(t)
	if err != nil {
		return
	}
	ctx := context.Background()
	col, err := d.db.Collection(ctx, COLLECTION_VERTICES)
	assert.NoError(t, err)

	meta, err := col.CreateDocument(ctx, map[string]interface{}{
		"_key":        "123",
		"description": "a",
	})
	assert.NoError(t, err, meta)
	meta, err = col.CreateDocument(ctx, map[string]interface{}{
		"_key":        "4",
		"description": "b",
	})
	assert.NoError(t, err, meta)

	graph, err := db.Graph(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, &model.Graph{
		Nodes: []*model.Node{
			{ID: "123"},
			{ID: "4"},
		},
		Edges: nil,
	}, graph)
}

func TestArangoDB_ValidateSchema(t *testing.T) {
	for _, test := range []struct {
		Name             string
		DBSetup          func(t *testing.T, db *ArangoDB)
		ExpError         bool
		ExpSchemaChanged bool
		ExpSchema        *driver.CollectionSchemaOptions
	}{
		{
			Name:             "empty db, should be NO-OP",
			DBSetup:          func(t *testing.T, db *ArangoDB) {},
			ExpSchemaChanged: false,
			ExpSchema:        &SchemaOptionsVertex,
			ExpError:         false,
		},
		{
			Name: "schema correct for all entries, should be NO-OP",
			DBSetup: func(t *testing.T, db *ArangoDB) {
				ctx := context.Background()
				col, err := db.db.Collection(ctx, COLLECTION_VERTICES)
				assert.NoError(t, err)
				meta, err := col.CreateDocument(ctx, map[string]interface{}{
					"_key":        "123",
					"description": "idk",
				})
				assert.NoError(t, err, meta)
			},
			ExpSchemaChanged: false,
			ExpSchema:        &SchemaOptionsVertex,
			ExpError:         false,
		},
		{
			Name: "schema updated (!= schema in code): new optional property -> compatible",
			DBSetup: func(t *testing.T, db *ArangoDB) {
				ctx := context.Background()
				assert := assert.New(t)
				col, err := db.db.Collection(ctx, COLLECTION_VERTICES)
				assert.NoError(err)
				meta, err := col.CreateDocument(ctx, map[string]interface{}{
					"_key":        "123",
					"description": "idk",
				})
				assert.NoError(err, meta)
				props, err := col.Properties(ctx)
				assert.NoError(err)
				props.Schema.Rule = copyMap(SchemaPropertyRulesVertice)
				props.Schema.Rule.(map[string]interface{})["properties"].(map[string]interface{})["newkey"] = map[string]string{
					"type": "string",
				}
				err = col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{Schema: props.Schema})
				assert.NoError(err)
			},
			ExpSchemaChanged: true,
			ExpSchema:        &SchemaOptionsVertex,
			ExpError:         false,
		},
		{
			Name: "schema updated (!= schema in code): new required property -> incompatible",
			DBSetup: func(t *testing.T, db *ArangoDB) {
				ctx := context.Background()
				assert := assert.New(t)
				col, err := db.db.Collection(ctx, COLLECTION_VERTICES)
				assert.NoError(err)
				meta, err := col.CreateDocument(ctx, map[string]interface{}{
					"_key":        "123",
					"description": "idk",
				})
				assert.NoError(err, meta)
				props, err := col.Properties(ctx)
				assert.NoError(err)
				props.Schema.Rule = copyMap(SchemaPropertyRulesVertice)
				props.Schema.Rule.(map[string]interface{})["properties"].(map[string]interface{})["newkey"] = map[string]string{
					"type": "string",
				}
				props.Schema.Rule.(map[string]interface{})["required"] = append(SchemaRequiredPropertiesVertice, "newkey")
				err = col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{Schema: props.Schema})
				assert.NoError(err)
			},
			ExpSchemaChanged: true,
			ExpSchema:        nil,
			ExpError:         true,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			_, db, err := dbTestSetupCleanup(t)
			if err != nil {
				return
			}
			test.DBSetup(t, db)
			ctx := context.Background()
			schemaChanged, err := db.ValidateSchema(ctx)
			assert := assert.New(t)
			assert.Equal(test.ExpSchemaChanged, schemaChanged)
			if test.ExpError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			col, err := db.db.Collection(ctx, COLLECTION_VERTICES)
			assert.NoError(err)
			props, err := col.Properties(ctx)
			assert.NoError(err)
			if test.ExpSchema != nil {
				assert.Equal(test.ExpSchema, props.Schema)
			}
		})
	}
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{})
	for k, v := range m {
		vm, ok := v.(map[string]interface{})
		if ok {
			cp[k] = copyMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}