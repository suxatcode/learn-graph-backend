//go:build integration

package db

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/suxatcode/learn-graph-poc-backend/graph/model"
	"github.com/suxatcode/learn-graph-poc-backend/middleware"
)

func init() {
	TESTONLY_initdb()
}

func testingSetupAndCleanupDB(t *testing.T) (DB, *ArangoDB, error) {
	db, err := NewArangoDB(TESTONLY_Config)
	assert.NoError(t, err)
	TESTONLY_SetupAndCleanup(t, db)
	return db, db.(*ArangoDB), err
}

func TestArangoDB_CreateDBWithSchema_HashIndexesOnUserCol(t *testing.T) {
	_, db, err := testingSetupAndCleanupDB(t)
	if err != nil {
		return
	}
	ctx := context.Background()
	assert := assert.New(t)
	col, err := db.db.Collection(ctx, COLLECTION_USERS)
	assert.NoError(err)
	indexes, err := col.Indexes(ctx)
	assert.NoError(err)
	assert.Len(indexes, 3)
	index_names := []string{}
	for _, index := range indexes {
		index_names = append(index_names, index.UserName())
	}
	assert.Contains(index_names, INDEX_HASH_USER_EMAIL)
	assert.Contains(index_names, INDEX_HASH_USER_USERNAME)
}

func TestArangoDB_CreateDBWithSchema_ExistingDBButMissingCol(t *testing.T) {
	_, db, err := testingSetupAndCleanupDB(t)
	if err != nil {
		return
	}
	ctx := context.Background()
	assert := assert.New(t)
	col, err := db.db.Collection(ctx, COLLECTION_USERS)
	assert.NoError(err)
	assert.NoError(col.Remove(ctx))
	assert.NoError(db.CreateDBWithSchema(ctx))
	exists, err := db.db.CollectionExists(ctx, COLLECTION_USERS)
	assert.NoError(err)
	assert.True(exists)
}

// Note: this setup is *inconsistent*, with actual data, since no corresponding
// node/edge edit-entries exist in COLLECTION_EDGEEDITS/COLLECTION_NODEEDITS!
func CreateNodesN0N1AndEdgeE0BetweenThem(t *testing.T, db *ArangoDB) {
	ctx := context.Background()
	col, err := db.db.Collection(ctx, COLLECTION_NODES)
	assert.NoError(t, err)
	meta, err := col.CreateDocument(ctx, map[string]interface{}{
		"_key":        "n0",
		"description": Text{"en": "a"},
	})
	assert.NoError(t, err, meta)
	meta, err = col.CreateDocument(ctx, map[string]interface{}{
		"_key":        "n1",
		"description": Text{"en": "b"},
	})
	assert.NoError(t, err, meta)
	col_edge, err := db.db.Collection(ctx, COLLECTION_EDGES)
	assert.NoError(t, err)
	meta, err = col_edge.CreateDocument(ctx, map[string]interface{}{
		"_key":   "e0",
		"_from":  fmt.Sprintf("%s/n0", COLLECTION_NODES),
		"_to":    fmt.Sprintf("%s/n1", COLLECTION_NODES),
		"weight": float64(2.0),
	})
	assert.NoError(t, err, meta)
}

func TestArangoDB_Graph(t *testing.T) {
	for _, test := range []struct {
		Name           string
		SetupDBContent func(t *testing.T, db *ArangoDB)
		ExpGraph       *model.Graph
		Context        context.Context
	}{
		{
			Name:    "2 nodes, no edges",
			Context: middleware.TestingCtxNewWithLanguage(context.Background(), "de"),
			SetupDBContent: func(t *testing.T, db *ArangoDB) {
				ctx := context.Background()
				col, err := db.db.Collection(ctx, COLLECTION_NODES)
				assert.NoError(t, err)
				meta, err := col.CreateDocument(ctx, map[string]interface{}{
					"_key":        "123",
					"description": Text{"de": "a"},
				})
				assert.NoError(t, err, meta)
				meta, err = col.CreateDocument(ctx, map[string]interface{}{
					"_key":        "4",
					"description": Text{"de": "b"},
				})
				assert.NoError(t, err, meta)
			},
			ExpGraph: &model.Graph{
				Nodes: []*model.Node{
					{ID: "123", Description: "a"},
					{ID: "4", Description: "b"},
				},
				Edges: nil,
			},
		},
		{
			Name:           "2 nodes, 1 edge",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			Context:        middleware.TestingCtxNewWithLanguage(context.Background(), "en"),
			ExpGraph: &model.Graph{
				Nodes: []*model.Node{
					{ID: "n0", Description: "a"},
					{ID: "n1", Description: "b"},
				},
				Edges: []*model.Edge{
					{
						ID:     "e0",
						From:   "n0",
						To:     "n1",
						Weight: float64(2.0),
					},
				},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db, d, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			test.SetupDBContent(t, d)

			graph, err := db.Graph(test.Context)
			assert.NoError(t, err)
			assert.Equal(t, test.ExpGraph, graph)
		})
	}
}

func TestArangoDB_AddEdgeWeightVote(t *testing.T) {
	for _, test := range []struct {
		Name                 string
		PreexistingUsers     []User
		PreexistingNodes     []Node
		PreexistingEdges     []Edge
		PreexistingEdgeEdits []EdgeEdit
		EdgeID               string
		EdgeWeight           float64
		ExpErr               bool
		ExpEdge              Edge
		ExpEdgeEdits         int
	}{
		{
			Name:   "err: no edge with id",
			EdgeID: "does-not-exist",
			ExpErr: true,
		},
		{
			Name: "edge found, weight averaged",
			PreexistingEdges: []Edge{
				{
					Document: Document{Key: "e0"},
					From:     fmt.Sprintf("%s/n0", COLLECTION_NODES),
					To:       fmt.Sprintf("%s/n1", COLLECTION_NODES),
					Weight:   2.0,
				},
			},
			PreexistingEdgeEdits: []EdgeEdit{
				{
					Edge:   "e0",
					User:   "u0",
					Type:   EdgeEditTypeCreate,
					Weight: 2.0,
				},
			},
			EdgeID:       "e0",
			ExpEdgeEdits: 2,
			EdgeWeight:   4.0,
			ExpErr:       false,
			ExpEdge:      Edge{Document: Document{Key: "e0"}, Weight: 3.0, From: fmt.Sprintf("%s/n0", COLLECTION_NODES), To: fmt.Sprintf("%s/n1", COLLECTION_NODES)},
		},
		{
			Name: "multiple votes exist, all shall be averaged",
			PreexistingEdges: []Edge{
				{
					Document: Document{Key: "e0"},
					From:     fmt.Sprintf("%s/n0", COLLECTION_NODES),
					To:       fmt.Sprintf("%s/n1", COLLECTION_NODES),
					Weight:   4.0,
				},
			},
			PreexistingEdgeEdits: []EdgeEdit{
				{
					Edge:   "e0",
					User:   "u0",
					Type:   EdgeEditTypeCreate,
					Weight: 2.0,
				},
				{
					Edge:   "e0",
					User:   "u0",
					Type:   EdgeEditTypeCreate,
					Weight: 6.0,
				},
			},
			EdgeID:       "e0",
			ExpEdgeEdits: 3,
			EdgeWeight:   10.0,
			ExpErr:       false,
			ExpEdge:      Edge{Document: Document{Key: "e0"}, Weight: 6.0, From: fmt.Sprintf("%s/n0", COLLECTION_NODES), To: fmt.Sprintf("%s/n1", COLLECTION_NODES)},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db, d, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if err := setupDBWithUsers(t, d, test.PreexistingUsers); err != nil {
				return
			}
			if err := setupDBWithGraph(t, d, test.PreexistingNodes, test.PreexistingEdges); err != nil {
				return
			}
			if err := setupDBWithEdits(t, d, []NodeEdit{}, test.PreexistingEdgeEdits); err != nil {
				return
			}
			err = db.AddEdgeWeightVote(context.Background(), User{Document: Document{Key: "321"}}, test.EdgeID, test.EdgeWeight)
			assert := assert.New(t)
			if test.ExpErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			ctx := context.Background()
			col, err := d.db.Collection(ctx, COLLECTION_EDGES)
			assert.NoError(err)
			e := Edge{}
			meta, err := col.ReadDocument(ctx, "e0", &e)
			assert.NoError(err, meta)
			assert.Equal(test.ExpEdge, e)
			edgeedits, err := QueryReadAll[EdgeEdit](ctx, d, `FOR e in edgeedits RETURN e`)
			assert.NoError(err)
			if !assert.Len(edgeedits, test.ExpEdgeEdits) {
				return
			}
			assert.Equal(edgeedits[len(edgeedits)-1].Edge, e.Key)
			assert.Equal(edgeedits[len(edgeedits)-1].User, "321")
			assert.Equal(edgeedits[len(edgeedits)-1].Type, EdgeEditTypeVote)
		})
	}
}

func TestArangoDB_CreateEdge(t *testing.T) {
	for _, test := range []struct {
		Name           string
		SetupDBContent func(t *testing.T, db *ArangoDB)
		From, To       string
		ExpErr         bool
	}{
		{
			Name:           "err: 'To' node-collection not found",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			From:           fmt.Sprintf("%s/n0", COLLECTION_NODES), To: "does-not-exist",
			ExpErr: true,
		},
		{
			Name:           "err: 'From' node-collection not found",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			From:           "does-not-exist", To: fmt.Sprintf("%s/n1", COLLECTION_NODES),
			ExpErr: true,
		},
		{
			Name:           "err: 'From' node-ID not found",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			From:           fmt.Sprintf("%s/doesnotexist", COLLECTION_NODES), To: fmt.Sprintf("%s/n1", COLLECTION_NODES),
			ExpErr: true,
		},
		{
			Name:           "err: 'To' node-ID not found",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			From:           fmt.Sprintf("%s/n1", COLLECTION_NODES), To: fmt.Sprintf("%s/doesnotexist", COLLECTION_NODES),
			ExpErr: true,
		},
		{
			Name:           "err: edge already exists",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			From:           fmt.Sprintf("%s/n0", COLLECTION_NODES), To: fmt.Sprintf("%s/n1", COLLECTION_NODES),
			ExpErr: true,
		},
		{
			Name:           "err: no self-linking nodes allowed",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			From:           fmt.Sprintf("%s/n0", COLLECTION_NODES), To: fmt.Sprintf("%s/n0", COLLECTION_NODES),
			ExpErr: true,
		},
		{
			Name:           "success: edge created and returned",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			From:           fmt.Sprintf("%s/n1", COLLECTION_NODES), To: fmt.Sprintf("%s/n0", COLLECTION_NODES),
			ExpErr: false,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db, d, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			test.SetupDBContent(t, d)
			weight := 1.1
			user123 := User{Document: Document{Key: "123"}}
			ID, err := db.CreateEdge(context.Background(), user123, test.From, test.To, weight)
			assert := assert.New(t)
			if test.ExpErr {
				assert.Error(err)
				assert.Empty(ID)
				return
			}
			assert.NoError(err)
			if !assert.NotEmptyf(ID, "edge ID: %v", err) {
				return
			}
			ctx := context.Background()
			col, err := d.db.Collection(ctx, COLLECTION_EDGES)
			assert.NoError(err)
			e := Edge{}
			meta, err := col.ReadDocument(ctx, ID, &e)
			if !assert.NoErrorf(err, "meta:%v,edge:%v,ID:'%s'", meta, e, ID) {
				return
			}
			assert.Equal(weight, e.Weight)
			edgeedits, err := QueryReadAll[EdgeEdit](ctx, d, `FOR e in edgeedits RETURN e`)
			assert.NoError(err)
			if !assert.Len(edgeedits, 1) {
				return
			}
			assert.Equal(edgeedits[0].Edge, e.Key)
			assert.Equal(edgeedits[0].User, user123.Key)
			assert.Equal(edgeedits[0].Type, EdgeEditTypeCreate)
			assert.Equal(edgeedits[0].Weight, weight)
		})
	}
}

func TestArangoDB_EditNode(t *testing.T) {
	for _, test := range []struct {
		Name           string
		SetupDBContent func(t *testing.T, db *ArangoDB)
		NodeID         string
		Description    *model.Text
		ExpError       bool
		ExpDescription Text
	}{
		{
			Name:           "err: node-ID not found",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			NodeID:         "does-not-exist",
			ExpError:       true,
		},
		{
			Name:           "success: description changed",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			NodeID:         "n0",
			Description: &model.Text{Translations: []*model.Translation{
				{Language: "en", Content: "new content"},
			}},
			ExpError:       false,
			ExpDescription: Text{"en": "new content"},
		},
		{
			Name:           "success: description merged different languages",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			NodeID:         "n0",
			Description: &model.Text{Translations: []*model.Translation{
				{Language: "ch", Content: "慈悲"},
			}},
			ExpError:       false,
			ExpDescription: Text{"en": "a", "ch": "慈悲"},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db, d, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			test.SetupDBContent(t, d)
			ctx := context.Background()
			err = db.EditNode(ctx, User{Document: Document{Key: "123"}}, test.NodeID, test.Description)
			assert := assert.New(t)
			if test.ExpError {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			col, err := d.db.Collection(ctx, COLLECTION_NODES)
			assert.NoError(err)
			node := Node{}
			meta, err := col.ReadDocument(ctx, test.NodeID, &node)
			assert.NoError(err, meta)
			assert.Equal(node.Description, test.ExpDescription)
			nodeedits, err := QueryReadAll[NodeEdit](ctx, d, `FOR e in nodeedits RETURN e`)
			assert.NoError(err)
			if !assert.Len(nodeedits, 1) {
				return
			}
			assert.Equal(test.NodeID, nodeedits[0].Node)
			assert.Equal("123", nodeedits[0].User)
			assert.Equal(NodeEditTypeEdit, nodeedits[0].Type)
			//assert.Equal(Text{"en": "a"}, nodeedits[0].NewNode.Description)
		})
	}
}

func TestArangoDB_ValidateSchema(t *testing.T) {
	addNewKeyToSchema := func(propertyRules map[string]interface{}, collection string) func(t *testing.T, db *ArangoDB) {
		return func(t *testing.T, db *ArangoDB) {
			ctx := context.Background()
			assert := assert.New(t)
			col, err := db.db.Collection(ctx, collection)
			assert.NoError(err)
			props, err := col.Properties(ctx)
			assert.NoError(err)
			if !assert.NotNil(props.Schema) {
				return
			}
			props.Schema.Rule = copyMap(propertyRules)
			props.Schema.Rule.(map[string]interface{})["properties"].(map[string]interface{})["newkey"] = map[string]string{
				"type": "string",
			}
			err = col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{Schema: props.Schema})
			assert.NoError(err)
		}
	}
	for _, test := range []struct {
		Name             string
		DBSetup          func(t *testing.T, db *ArangoDB)
		ExpError         bool
		ExpSchemaChanged SchemaUpdateAction
		ExpNodeSchema    *driver.CollectionSchemaOptions
	}{
		{
			Name:             "empty db, should be NO-OP",
			DBSetup:          func(t *testing.T, db *ArangoDB) {},
			ExpSchemaChanged: SchemaUnchanged,
			ExpNodeSchema:    &SchemaOptionsNode,
			ExpError:         false,
		},
		{
			Name: "schema correct for all entries, should be NO-OP",
			DBSetup: func(t *testing.T, db *ArangoDB) {
				ctx := context.Background()
				col, err := db.db.Collection(ctx, COLLECTION_NODES)
				assert.NoError(t, err)
				meta, err := col.CreateDocument(ctx, map[string]interface{}{
					"_key":        "123",
					"description": Text{"en": "idk"},
				})
				assert.NoError(t, err, meta)
			},
			ExpSchemaChanged: SchemaUnchanged,
			ExpNodeSchema:    &SchemaOptionsNode,
			ExpError:         false,
		},
		{
			Name: "schema updated (!= schema in code): new optional property -> compatible",
			DBSetup: func(t *testing.T, db *ArangoDB) {
				ctx := context.Background()
				assert := assert.New(t)
				col, err := db.db.Collection(ctx, COLLECTION_NODES)
				assert.NoError(err)
				meta, err := col.CreateDocument(ctx, map[string]interface{}{
					"_key":        "123",
					"description": Text{"en": "idk"},
				})
				assert.NoError(err, meta)
				props, err := col.Properties(ctx)
				assert.NoError(err)
				props.Schema.Rule = copyMap(SchemaPropertyRulesNode)
				props.Schema.Rule.(map[string]interface{})["properties"].(map[string]interface{})["newkey"] = map[string]string{
					"type": "string",
				}
				err = col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{Schema: props.Schema})
				assert.NoError(err)
			},
			ExpSchemaChanged: SchemaChangedButNoActionRequired,
			ExpNodeSchema:    &SchemaOptionsNode,
			ExpError:         false,
		},
		{
			Name: "schema updated (!= schema in code): new required property -> incompatible",
			DBSetup: func(t *testing.T, db *ArangoDB) {
				ctx := context.Background()
				assert := assert.New(t)
				col, err := db.db.Collection(ctx, COLLECTION_NODES)
				assert.NoError(err)
				meta, err := col.CreateDocument(ctx, map[string]interface{}{
					"_key":        "123",
					"description": Text{"en": "idk"},
				})
				assert.NoError(err, meta)
				props, err := col.Properties(ctx)
				assert.NoError(err)
				props.Schema.Rule = copyMap(SchemaPropertyRulesNode)
				props.Schema.Rule.(map[string]interface{})["properties"].(map[string]interface{})["newkey"] = map[string]string{
					"type": "string",
				}
				props.Schema.Rule.(map[string]interface{})["required"] = append(SchemaPropertyRulesNode["required"].([]interface{}), "newkey")
				err = col.SetProperties(ctx, driver.SetCollectionPropertiesOptions{Schema: props.Schema})
				assert.NoError(err)
			},
			ExpSchemaChanged: SchemaChangedButNoActionRequired,
			ExpNodeSchema:    nil,
			ExpError:         true,
		},
		{
			Name:             "collection users should be verified",
			DBSetup:          addNewKeyToSchema(SchemaPropertyRulesUser, COLLECTION_USERS),
			ExpSchemaChanged: SchemaChangedButNoActionRequired,
			ExpNodeSchema:    nil,
			ExpError:         false,
		},
		{
			Name:             "collection nodeedits should be verified",
			DBSetup:          addNewKeyToSchema(SchemaPropertyRulesNodeEdit, COLLECTION_NODEEDITS),
			ExpSchemaChanged: SchemaChangedButNoActionRequired,
			ExpNodeSchema:    nil,
			ExpError:         false,
		},
		{
			Name:             "collection edgeedits should be verified",
			DBSetup:          addNewKeyToSchema(SchemaPropertyRulesEdgeEdit, COLLECTION_EDGEEDITS),
			ExpSchemaChanged: SchemaChangedButNoActionRequired,
			ExpNodeSchema:    nil,
			ExpError:         false,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
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
			if test.ExpNodeSchema != nil {
				nodeCol, err := db.db.Collection(ctx, COLLECTION_NODES)
				assert.NoError(err)
				nodeProps, err := nodeCol.Properties(ctx)
				assert.NoError(err)
				assert.Equal(test.ExpNodeSchema, nodeProps.Schema)
			}
		})
	}
}

// recursive map copy
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

func TestArangoDB_CreateNode(t *testing.T) {
	for _, test := range []struct {
		Name         string
		Translations []*model.Translation
		User         User
		ExpError     bool
	}{
		{
			Name: "single translation: language 'en'",
			User: User{Document: Document{Key: "123"}},
			Translations: []*model.Translation{
				{Language: "en", Content: "abc"},
			},
		},
		{
			Name: "multiple translations: language 'en', 'de', 'ch'",
			Translations: []*model.Translation{
				{Language: "en", Content: "Hello World!"},
				{Language: "de", Content: "Hallo Welt!"},
				{Language: "ch", Content: "你好世界！"},
			},
		},
		{
			Name: "invalid translation language",
			Translations: []*model.Translation{
				{Language: "AAAAA", Content: "idk"},
			},
			ExpError: true,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			ctx := context.Background()
			assert := assert.New(t)
			id, err := db.CreateNode(ctx, test.User, &model.Text{
				Translations: test.Translations,
			})
			if test.ExpError {
				assert.Error(err)
				assert.Equal("", id)
				return
			}
			assert.NoError(err)
			assert.NotEqual("", id)
			nodes, err := QueryReadAll[Node](ctx, db, `FOR n in nodes RETURN n`)
			assert.NoError(err)
			t.Logf("id: %v, nodes: %#v", id, nodes)
			text := Text{}
			for lang, content := range ConvertToDBText(&model.Text{Translations: test.Translations}) {
				text[lang] = content
			}
			n := FindFirst(nodes, func(n Node) bool {
				return reflect.DeepEqual(n.Description, text)
			})
			if !assert.NotNil(n) {
				return
			}
			nodeedits, err := QueryReadAll[NodeEdit](ctx, db, `FOR e in nodeedits RETURN e`)
			assert.NoError(err)
			if !assert.Len(nodeedits, 1) {
				return
			}
			assert.Equal(n.Key, nodeedits[0].Node)
		})
	}
}

func strptr(s string) *string {
	return &s
}

func TestArangoDB_verifyUserInput(t *testing.T) {
	for _, test := range []struct {
		TestName      string
		User          User
		Password      string
		ExpectNil     bool
		Result        model.CreateUserResult
		ExistingUsers []User
	}{
		{
			TestName:  "duplicate username",
			User:      User{Username: "abcd", EMail: "abc@def.com"},
			Password:  "1234567890",
			ExpectNil: false,
			Result: model.CreateUserResult{
				Login: &model.LoginResult{
					Success: false,
					Message: strptr("Username already exists: 'abcd'"),
				},
			},
			ExistingUsers: []User{
				{
					Username:     "abcd",
					EMail:        "old@mail.com",
					PasswordHash: "$2a$10$UuEBwAF9YQ2OYgTZ9qy8Oeh04HkWcC3S/P4680pz7tII.wnGc0U0y",
				},
			},
		},
		{
			TestName:  "duplicate email",
			User:      User{Username: "abcd", EMail: "abc@def.com"},
			Password:  "1234567890",
			ExpectNil: false,
			Result: model.CreateUserResult{
				Login: &model.LoginResult{
					Success: false,
					Message: strptr("EMail already exists: 'abc@def.com'"),
				},
			},
			ExistingUsers: []User{
				{
					Username:     "mrxx",
					EMail:        "abc@def.com",
					PasswordHash: "$2a$10$UuEBwAF9YQ2OYgTZ9qy8Oeh04HkWcC3S/P4680pz7tII.wnGc0U0y",
				},
			},
		},
	} {
		t.Run(test.TestName, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if len(test.ExistingUsers) >= 1 {
				if err := setupDBWithUsers(t, db, test.ExistingUsers); err != nil {
					return
				}
			}
			ctx := context.Background()
			assert := assert.New(t)
			res, err := db.verifyUserInput(ctx, test.User, test.Password)
			assert.NoError(err)
			if test.ExpectNil {
				assert.Nil(res)
				return
			}
			if !assert.NotNil(res) {
				return
			}
			assert.Equal(test.Result.Login.Success, res.Login.Success)
			if test.Result.Login.Message != nil {
				assert.Equal(*test.Result.Login.Message, *res.Login.Message)
			}
		})
	}
}

func TestArangoDB_createUser(t *testing.T) {
	for _, test := range []struct {
		TestName      string
		User          User
		Password      string
		ExpectError   bool
		Result        model.CreateUserResult
		ExistingUsers []User
	}{
		{
			TestName:    "username already exists",
			User:        User{Username: "mrxx", EMail: "new@def.com"},
			Password:    "1234567890",
			ExpectError: true,
			ExistingUsers: []User{
				{
					Username:     "mrxx",
					EMail:        "abc@def.com",
					PasswordHash: "$2a$10$UuEBwAF9YQ2OYgTZ9qy8Oeh04HkWcC3S/P4680pz7tII.wnGc0U0y",
				},
			},
		},
		{
			TestName: "a user with that email exists already",
			User: User{Username: "mrxx",
				EMail: "abc@def.com"},
			Password:    "1234567890",
			ExpectError: true,
			ExistingUsers: []User{
				{
					Username:     "abcd",
					EMail:        "abc@def.com",
					PasswordHash: "$2a$10$UuEBwAF9YQ2OYgTZ9qy8Oeh04HkWcC3S/P4680pz7tII.wnGc0U0y",
				},
			},
		},
	} {
		t.Run(test.TestName, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if len(test.ExistingUsers) >= 1 {
				if err := setupDBWithUsers(t, db, test.ExistingUsers); err != nil {
					return
				}
			}
			ctx := context.Background()
			_, err = db.createUser(ctx, test.User, test.Password)
			assert := assert.New(t)
			if test.ExpectError {
				assert.Error(err)
				return
			}
			assert.True(false) // should never be reached
		})
	}
}

func TestArangoDB_CreateUserWithEMail(t *testing.T) {
	for _, test := range []struct {
		TestName                  string
		UserName, Password, EMail string
		ExpectError               bool
		Result                    model.CreateUserResult
		ExistingUsers             []User
	}{
		{
			TestName: "valid everything",
			UserName: "abcd",
			Password: "1234567890",
			EMail:    "abc@def.com",
			Result: model.CreateUserResult{
				Login: &model.LoginResult{
					Success: true,
				},
			},
		},
		{
			// MAYBE: https://github.com/wagslane/go-password-validator, or just 2FA
			TestName: "password too small: < MIN_PASSWORD_LENGTH characters",
			UserName: "abcd",
			Password: "123456789",
			EMail:    "abc@def.com",
			Result: model.CreateUserResult{
				Login: &model.LoginResult{
					Success: false,
					Message: strptr("Password must be at least length"),
				},
			},
		},
		{
			TestName: "username too small: < MIN_USERNAME_LENGTH characters",
			UserName: "o.o",
			Password: "1234567890",
			EMail:    "abc@def.com",
			Result: model.CreateUserResult{
				Login: &model.LoginResult{
					Success: false,
					Message: strptr("Username must be at least length"),
				},
			},
		},
		{
			TestName: "invalid email",
			UserName: "abcd",
			Password: "1234567890",
			EMail:    "abc@def@com",
			Result: model.CreateUserResult{
				Login: &model.LoginResult{
					Success: false,
					Message: strptr("Invalid EMail"),
				},
			},
		},
	} {
		t.Run(test.TestName, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if len(test.ExistingUsers) >= 1 {
				if err := setupDBWithUsers(t, db, test.ExistingUsers); err != nil {
					return
				}
			}
			ctx := context.Background()
			res, err := db.CreateUserWithEMail(ctx, test.UserName, test.Password, test.EMail)
			assert := assert.New(t)
			if test.ExpectError {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			users, err := QueryReadAll[User](ctx, db, `FOR u in users RETURN u`)
			assert.NoError(err)
			if !assert.Equal(test.Result.Login.Success, res.Login.Success, "unexpected login result") {
				return
			}
			if !test.Result.Login.Success {
				assert.Contains(*res.Login.Message, *test.Result.Login.Message)
				assert.Empty(res.Login.UserID, "there should not be a user ID, if creation fails")
				assert.Empty(users, "there should be no users in DB")
				return
			}
			assert.NotEmpty(res.Login.UserID)
			if !assert.Len(users, 1, "one user should be created in DB") {
				return
			}
			assert.Equal(test.UserName, users[0].Username)
			assert.Equal(test.UserName, res.Login.UserName)
			assert.Equal(users[0].Document.Key, res.Login.UserID)
			if !assert.NotEmpty(res.Login.Token, "login token should be returned") {
				return
			}
			_, err = uuid.Parse(res.Login.Token)
			assert.NoError(err)
			assert.Len(users[0].Tokens, 1, "there should be one token in DB")
			assert.Equal(users[0].Tokens[0].Token, res.Login.Token)
		})
	}
}

func setupCollectionWithDocuments[T any](t *testing.T, db *ArangoDB, collection string, documents []T) error {
	ctx := context.Background()
	col, err := db.db.Collection(ctx, collection)
	if !assert.NoError(t, err) {
		return err
	}
	for _, doc := range documents {
		meta, err := col.CreateDocument(ctx, doc)
		if !assert.NoError(t, err, meta) {
			return err
		}
	}
	return nil
}

func setupDBWithUsers(t *testing.T, db *ArangoDB, users []User) error {
	return setupCollectionWithDocuments(t, db, COLLECTION_USERS, users)
}

func setupDBWithGraph(t *testing.T, db *ArangoDB, nodes []Node, edges []Edge) error {
	err := setupCollectionWithDocuments(t, db, COLLECTION_NODES, nodes)
	if err != nil {
		return err
	}
	return setupCollectionWithDocuments(t, db, COLLECTION_EDGES, edges)
}

func setupDBWithEdits(t *testing.T, db *ArangoDB, nodeedits []NodeEdit, edgeedits []EdgeEdit) error {
	err := setupCollectionWithDocuments(t, db, COLLECTION_NODEEDITS, nodeedits)
	if err != nil {
		return err
	}
	return setupCollectionWithDocuments(t, db, COLLECTION_EDGEEDITS, edgeedits)
}

func TestArangoDB_Login(t *testing.T) {
	for _, test := range []struct {
		TestName              string
		Auth                  model.LoginAuthentication
		ExpectError           bool
		ExpectLoginSuccess    bool
		ExpectErrorMessage    string
		ExistingUsers         []User
		TokenAmountAfterLogin int
		//Result                model.LoginResult
	}{
		{
			TestName: "user does not exist",
			Auth: model.LoginAuthentication{
				Email:    "abc@def.com",
				Password: "1234567890",
			},
			ExpectError:        false,
			ExpectLoginSuccess: false,
			ExpectErrorMessage: "User does not exist",
		},
		{
			TestName: "password missmatch",
			Auth: model.LoginAuthentication{
				Email:    "abc@def.com",
				Password: "1234567890",
			},
			ExpectError:        false,
			ExpectLoginSuccess: false,
			ExpectErrorMessage: "Password missmatch",
			ExistingUsers: []User{
				{
					Username:     "abcd",
					EMail:        "abc@def.com",
					PasswordHash: "$2a$10$UAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAI.wnGc0U0y",
				},
			},
		},
		{
			TestName: "successful login",
			Auth: model.LoginAuthentication{
				Email:    "abc@def.com",
				Password: "1234567890",
			},
			ExpectError:        false,
			ExpectLoginSuccess: true,
			ExpectErrorMessage: "",
			ExistingUsers: []User{
				{
					Username:     "abcd",
					EMail:        "abc@def.com",
					PasswordHash: "$2a$10$UuEBwAF9YQ2OYgTZ9qy8Oeh04HkWcC3S/P4680pz7tII.wnGc0U0y",
				},
			},
			TokenAmountAfterLogin: 1,
		},
	} {
		t.Run(test.TestName, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if len(test.ExistingUsers) >= 1 {
				if err := setupDBWithUsers(t, db, test.ExistingUsers); err != nil {
					return
				}
			}
			ctx := context.Background()
			res, err := db.Login(ctx, test.Auth)
			assert := assert.New(t)
			if test.ExpectError {
				assert.Error(err)
				assert.Nil(res)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.ExpectLoginSuccess, res.Success)
			if !test.ExpectLoginSuccess {
				assert.False(res.Success)
				assert.Empty(res.Token)
				assert.Contains(*res.Message, test.ExpectErrorMessage)
				return
			}
			if !assert.NotEmpty(res.Token, "login token should be returned") {
				return
			}
			assert.NotEmpty(res.UserID)
			_, err = uuid.Parse(res.Token)
			assert.NoError(err)
			users, err := QueryReadAll[User](ctx, db, `FOR u in users FILTER u.email == @name RETURN u`, map[string]interface{}{
				"name": test.Auth.Email,
			})
			assert.NoError(err)
			assert.Len(users, 1)
			user := users[0]
			if !assert.Len(user.Tokens, test.TokenAmountAfterLogin) {
				return
			}
			assert.Equal(user.Tokens[test.TokenAmountAfterLogin-1].Token, res.Token)
			assert.Equal(user.Username, res.UserName)
			assert.Equal(user.Document.Key, res.UserID)
			_, err = uuid.Parse(user.Tokens[test.TokenAmountAfterLogin-1].Token)
			assert.NoError(err)
		})
	}
}

func TestArangoDB_deleteUserByKey(t *testing.T) {
	for _, test := range []struct {
		TestName, KeyToDelete string
		PreexistingUsers      []User
		MakeCtxFn             func(ctx context.Context) context.Context
		ExpectError           bool
	}{
		{
			TestName:    "successful deletion",
			KeyToDelete: "123",
			PreexistingUsers: []User{
				{
					Document:     Document{Key: "123"},
					Username:     "abcd",
					EMail:        "a@b.com",
					PasswordHash: "321",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(24 * time.Hour).UnixMilli(), Token: "TOKEN"},
					},
				},
			},
			MakeCtxFn: func(ctx context.Context) context.Context {
				return middleware.TestingCtxNewWithAuthentication(ctx, "TOKEN")
			},
		},
		{
			TestName:    "error: token expired",
			KeyToDelete: "123",
			PreexistingUsers: []User{
				{
					Document:     Document{Key: "123"},
					Username:     "abcd",
					EMail:        "a@b.com",
					PasswordHash: "321",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(-24 * time.Hour).UnixMilli(), Token: "TOKEN"},
					},
				},
			},
			MakeCtxFn: func(ctx context.Context) context.Context {
				return middleware.TestingCtxNewWithAuthentication(ctx, "TOKEN")
			},
			ExpectError: true,
		},
		{
			TestName:    "error: no such user ID",
			KeyToDelete: "1",
			ExpectError: true,
		},
		{
			TestName:    "error: no matching auth token for user ID",
			KeyToDelete: "123",
			PreexistingUsers: []User{
				{
					Document:     Document{Key: "123"},
					Username:     "abcd",
					EMail:        "a@b.com",
					PasswordHash: "321",
					Tokens: []AuthenticationToken{
						{Token: "AAAAA"},
					},
				},
			},
			MakeCtxFn: func(ctx context.Context) context.Context {
				return middleware.TestingCtxNewWithAuthentication(ctx, "BBBBBB")
			},
			ExpectError: true,
		},
	} {
		t.Run(test.TestName, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if err := setupDBWithUsers(t, db, test.PreexistingUsers); err != nil {
				return
			}
			ctx := context.Background()
			if test.MakeCtxFn != nil {
				ctx = test.MakeCtxFn(ctx)
			}
			assert := assert.New(t)
			err = db.deleteUserByKey(ctx, test.KeyToDelete)
			if test.ExpectError {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}

var ralf = User{
	Document:     Document{Key: "123"},
	Username:     "ralf",
	EMail:        "a@b.com",
	PasswordHash: "321",
	Tokens: []AuthenticationToken{
		{Token: "TOKEN"},
	},
}

func TestArangoDB_getUserByProperty(t *testing.T) {
	for _, test := range []struct {
		TestName, Property, Value string
		PreexistingUsers          []User
		ExpectError               bool
		ExpectedResult            *User
	}{
		{
			TestName: "retrieve existing user by username successfully",
			Property: "username",
			Value:    "ralf",
			PreexistingUsers: []User{
				ralf,
			},
			ExpectedResult: &ralf,
		},
		{
			TestName:         "error: no such user",
			Property:         "username",
			Value:            "ralf",
			PreexistingUsers: []User{},
		},
	} {
		t.Run(test.TestName, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if err := setupDBWithUsers(t, db, test.PreexistingUsers); err != nil {
				return
			}
			ctx := context.Background()
			user, err := db.getUserByProperty(ctx, test.Property, test.Value)
			assert := assert.New(t)
			if test.ExpectError {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(test.ExpectedResult, user)
		})
	}
}

func TestArangoDB_DeleteAccount(t *testing.T) {
	for _, test := range []struct {
		TestName         string
		PreexistingUsers []User
		MakeCtxFn        func(ctx context.Context) context.Context
		ExpectError      bool
	}{
		{
			TestName: "successful deletion",
			PreexistingUsers: []User{
				{
					Document:     Document{Key: "abcd"},
					Username:     "lmas",
					EMail:        "a@b.com",
					PasswordHash: "321",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(24 * time.Hour).UnixMilli(), Token: "TOKEN"},
					},
				},
			},
			MakeCtxFn: func(ctx context.Context) context.Context {
				ctx = middleware.TestingCtxNewWithAuthentication(ctx, "TOKEN")
				ctx = middleware.TestingCtxNewWithUserID(ctx, "abcd")
				return ctx
			},
		},
		{
			TestName: "error: no such user for _key",
			MakeCtxFn: func(ctx context.Context) context.Context {
				return middleware.TestingCtxNewWithUserID(ctx, "abcd")
			},
			ExpectError: true,
		},
		{
			TestName: "error: no matching auth token for user._key",
			PreexistingUsers: []User{
				{
					Document:     Document{Key: "abcd"},
					Username:     "lmas",
					EMail:        "a@b.com",
					PasswordHash: "321",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(24 * time.Hour).UnixMilli(), Token: "AAAAA"},
					},
				},
			},
			MakeCtxFn: func(ctx context.Context) context.Context {
				ctx = middleware.TestingCtxNewWithAuthentication(ctx, "BBBBBB")
				ctx = middleware.TestingCtxNewWithUserID(ctx, "abcd")
				return ctx
			},
			ExpectError: true,
		},
	} {
		t.Run(test.TestName, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if err := setupDBWithUsers(t, db, test.PreexistingUsers); err != nil {
				return
			}
			ctx := context.Background()
			if test.MakeCtxFn != nil {
				ctx = test.MakeCtxFn(ctx)
			}
			assert := assert.New(t)
			err = db.DeleteAccount(ctx)
			if test.ExpectError {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}

func TestArangoDB_Logout(t *testing.T) {
	for _, test := range []struct {
		Name                            string
		ContextUserID, ContextAuthToken string
		PreexistingUsers                []User
		ExpErr                          bool
		ExpTokenLenAfterLogout          int
	}{
		{
			Name:             "successful logout",
			ContextUserID:    "123",
			ContextAuthToken: "TOKEN",
			PreexistingUsers: []User{
				{
					Document:     Document{Key: "123"},
					Username:     "abcd",
					EMail:        "a@b.com",
					PasswordHash: "321",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(24 * time.Hour).UnixMilli(), Token: "TOKEN"},
					},
				},
			},
			ExpErr:                 false,
			ExpTokenLenAfterLogout: 0,
		},
		{
			Name:             "fail: token missmatch",
			ContextUserID:    "456",
			ContextAuthToken: "AAA",
			PreexistingUsers: []User{
				{
					Document:     Document{Key: "456"},
					Username:     "abcd",
					EMail:        "a@b.com",
					PasswordHash: "321",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(24 * time.Hour).UnixMilli(), Token: "BBB"},
					},
				},
			},
			ExpErr:                 true,
			ExpTokenLenAfterLogout: 1,
		},
		{
			Name:             "success: token expired, but doesn't matter user wants to remove it anyways",
			ContextUserID:    "456",
			ContextAuthToken: "TOKEN",
			PreexistingUsers: []User{
				{
					Document:     Document{Key: "456"},
					Username:     "abcd",
					EMail:        "a@b.com",
					PasswordHash: "321",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(-24 * time.Hour).UnixMilli(), Token: "TOKEN"},
					},
				},
			},
			ExpErr:                 false,
			ExpTokenLenAfterLogout: 0,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if err := setupDBWithUsers(t, db, test.PreexistingUsers); err != nil {
				return
			}
			ctx := middleware.TestingCtxNewWithUserID(context.Background(), test.ContextUserID)
			ctx = middleware.TestingCtxNewWithAuthentication(ctx, test.ContextAuthToken)
			assert := assert.New(t)
			err = db.Logout(ctx)
			if test.ExpErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			users, err := QueryReadAll[User](ctx, db, `FOR u in users FILTER u._key == @key RETURN u`, map[string]interface{}{
				"key": test.ContextUserID,
			})
			if len(users) != 1 {
				return
			}
			assert.NoError(err)
			assert.Len(users[0].Tokens, test.ExpTokenLenAfterLogout)
		})
	}
}

func TestArangoDB_IsUserAuthenticated(t *testing.T) {
	for _, test := range []struct {
		Name                            string
		ContextUserID, ContextAuthToken string
		PreexistingUsers                []User
		ExpErr                          bool
		ExpAuth                         bool
	}{
		{
			Name:             "userID not found",
			ContextUserID:    "qwerty",
			ContextAuthToken: "123",
			ExpErr:           false,
			ExpAuth:          false,
		},
		{
			Name:             "userID found, but no valid token",
			ContextUserID:    "qwerty",
			ContextAuthToken: "AAA",
			PreexistingUsers: []User{
				{
					Document: Document{Key: "qwerty"},
					Username: "asdf",
					EMail:    "a@b.com",
					Tokens: []AuthenticationToken{
						{Token: "BBB"},
					},
				},
			},
			ExpErr:  false,
			ExpAuth: false,
		},
		{
			Name:             "user authenticated, everything valid",
			ContextUserID:    "qwerty",
			ContextAuthToken: "AAA",
			PreexistingUsers: []User{
				{
					Document: Document{Key: "qwerty"},
					Username: "abcd",
					EMail:    "a@b.com",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(24 * time.Hour).UnixMilli(), Token: "AAA"},
					},
				},
			},
			ExpErr:  false,
			ExpAuth: true,
		},
		{
			Name:             "user authenticated, matching token, but expired",
			ContextUserID:    "qwerty",
			ContextAuthToken: "AAA",
			PreexistingUsers: []User{
				{
					Document: Document{Key: "qwerty"},
					Username: "abcd",
					EMail:    "a@b.com",
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(-24 * time.Hour).UnixMilli(), Token: "AAA"},
					},
				},
			},
			ExpErr:  false,
			ExpAuth: false,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if err := setupDBWithUsers(t, db, test.PreexistingUsers); err != nil {
				return
			}
			ctx := middleware.TestingCtxNewWithUserID(context.Background(), test.ContextUserID)
			ctx = middleware.TestingCtxNewWithAuthentication(ctx, test.ContextAuthToken)
			auth, user, err := db.IsUserAuthenticated(ctx)
			assert := assert.New(t)
			assert.Equal(test.ExpAuth, auth)
			if test.ExpErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			if test.ExpAuth {
				assert.NotNil(user)
			}
		})
	}
}

func TestArangoDB_DeleteAccountWithData(t *testing.T) {
	for _, test := range []struct {
		Name                 string
		UsernameToDelete     string
		UsersLeftOver        int
		ContextUserID        string
		ContextAuthToken     string
		ExpectError          bool
		PreexistingUsers     []User
		PreexistingNodes     []Node
		PreexistingNodeEdits []NodeEdit
		PreexistingEdges     []Edge
		PreexistingEdgeEdits []EdgeEdit
	}{
		{
			Name:             "deletion of single node",
			UsernameToDelete: "asdf",
			UsersLeftOver:    1,
			ContextUserID:    "hasadmin",
			ContextAuthToken: "AAA",
			PreexistingUsers: []User{
				{
					Document: Document{Key: "1"},
					Username: "asdf",
					EMail:    "a@b.com",
				},
				{
					Document: Document{Key: "hasadmin"},
					Username: "qwerty",
					EMail:    "d@e.com",
					Roles:    []RoleType{RoleAdmin},
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(24 * time.Hour).UnixMilli(), Token: "AAA"},
					},
				},
			},
			PreexistingNodes: []Node{
				{
					Document:    Document{Key: "2"},
					Description: Text{"en": "hello"},
				},
			},
			PreexistingNodeEdits: []NodeEdit{
				{
					Document: Document{Key: "3"},
					Node:     "2",
					User:     "1",
					Type:     NodeEditTypeCreate,
					NewNode: Node{
						Document:    Document{Key: "2"},
						Description: Text{"en": "hello"},
					},
				},
			},
		},
		{
			Name:             "user has no admin role -> expect failure",
			UsernameToDelete: "asdf",
			UsersLeftOver:    1,
			ContextUserID:    "2",
			ContextAuthToken: "AAA",
			ExpectError:      true,
			PreexistingUsers: []User{
				{
					Document: Document{Key: "1"},
					Username: "asdf",
					EMail:    "a@b.com",
				},
				{
					Document: Document{Key: "2"},
					Username: "qwerty",
					EMail:    "d@e.com",
					Roles:    []RoleType{ /*empty!*/ },
					Tokens: []AuthenticationToken{
						{Expiry: time.Now().Add(24 * time.Hour).UnixMilli(), Token: "AAA"},
					},
				},
			},
			PreexistingNodes: []Node{
				{
					Document:    Document{Key: "2"},
					Description: Text{"en": "hello"},
				},
			},
			PreexistingNodeEdits: []NodeEdit{
				{
					Document: Document{Key: "3"},
					Node:     "2",
					User:     "1",
					Type:     NodeEditTypeCreate,
					NewNode: Node{
						Document:    Document{Key: "2"},
						Description: Text{"en": "hello"},
					},
				},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			_, db, err := testingSetupAndCleanupDB(t)
			if err != nil {
				return
			}
			if err := setupDBWithUsers(t, db, test.PreexistingUsers); err != nil {
				return
			}
			if err := setupDBWithGraph(t, db, test.PreexistingNodes, test.PreexistingEdges); err != nil {
				return
			}
			if err := setupDBWithEdits(t, db, test.PreexistingNodeEdits, test.PreexistingEdgeEdits); err != nil {
				return
			}
			ctx := middleware.TestingCtxNewWithUserID(context.Background(), test.ContextUserID)
			ctx = middleware.TestingCtxNewWithAuthentication(ctx, test.ContextAuthToken)
			err = db.DeleteAccountWithData(ctx, test.UsernameToDelete, "1234")
			assert := assert.New(t)
			if test.ExpectError {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			users, err := QueryReadAll[User](ctx, db, `FOR u in users RETURN u`)
			assert.NoError(err)
			assert.Len(users, test.UsersLeftOver)
		})
	}
}
