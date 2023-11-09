//go:build integration

package db

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/arangodb/go-driver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/suxatcode/learn-graph-poc-backend/graph/model"
	"github.com/suxatcode/learn-graph-poc-backend/middleware"
)

var testConfig = Config{
	Host:             "http://localhost:18529",
	NoAuthentication: true,
}

func TestNewArangoDB(t *testing.T) {
	_, err := NewArangoDB(testConfig)
	assert.NoError(t, err, "expected connection succeeds")
}

func SetupDB(db *ArangoDB, t *testing.T) error {
	return db.CreateDBWithSchema(context.Background())
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
	err = SetupDB(db.(*ArangoDB), t)
	assert.NoError(t, err)
	return db, db.(*ArangoDB), err
}

func TestArangoDB_CreateDBWithSchema(t *testing.T) {
	_, db, err := dbTestSetupCleanup(t)
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
		"weight": float64(3.141),
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
			Context: middleware.CtxNewWithLanguage(context.Background(), "de"),
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
			Context:        middleware.CtxNewWithLanguage(context.Background(), "en"),
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
						Weight: float64(3.141),
					},
				},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db, d, err := dbTestSetupCleanup(t)
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

func TestArangoDB_SetEdgeWeight(t *testing.T) {
	for _, test := range []struct {
		Name           string
		SetupDBContent func(t *testing.T, db *ArangoDB)
		EdgeID         string
		EdgeWeight     float64
		ExpErr         bool
		ExpEdge        Edge
	}{
		{
			Name:           "err: no edge with id",
			SetupDBContent: func(t *testing.T, db *ArangoDB) { /*empty db*/ },
			EdgeID:         "does-not-exist",
			ExpErr:         true,
		},
		{
			Name:           "edge found, weight changed",
			SetupDBContent: CreateNodesN0N1AndEdgeE0BetweenThem,
			EdgeID:         "e0",
			EdgeWeight:     9.9,
			ExpErr:         false,
			ExpEdge:        Edge{Document: Document{Key: "e0"}, Weight: 9.9, From: fmt.Sprintf("%s/n0", COLLECTION_NODES), To: fmt.Sprintf("%s/n1", COLLECTION_NODES)},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db, d, err := dbTestSetupCleanup(t)
			if err != nil {
				return
			}
			test.SetupDBContent(t, d)
			err = db.SetEdgeWeight(context.Background(), test.EdgeID, test.EdgeWeight)
			if test.ExpErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			ctx := context.Background()
			col, err := d.db.Collection(ctx, COLLECTION_EDGES)
			assert.NoError(t, err)
			e := Edge{}
			meta, err := col.ReadDocument(ctx, "e0", &e)
			assert.NoError(t, err, meta)
			assert.Equal(t, test.ExpEdge, e)
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
			db, d, err := dbTestSetupCleanup(t)
			if err != nil {
				return
			}
			test.SetupDBContent(t, d)
			weight := 1.1
			ID, err := db.CreateEdge(context.Background(), test.From, test.To, weight)
			if test.ExpErr {
				assert.Error(t, err)
				assert.Empty(t, ID)
				return
			}
			assert.NoError(t, err)
			if !assert.NotEmpty(t, ID, "edge ID: %v", err) {
				return
			}
			ctx := context.Background()
			col, err := d.db.Collection(ctx, COLLECTION_EDGES)
			assert.NoError(t, err)
			e := Edge{}
			meta, err := col.ReadDocument(ctx, ID, &e)
			assert.NoErrorf(t, err, "meta:%v,edge:%v", meta, e)
			assert.Equal(t, weight, e.Weight)
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
			db, d, err := dbTestSetupCleanup(t)
			if err != nil {
				return
			}
			test.SetupDBContent(t, d)
			ctx := context.Background()
			err = db.EditNode(ctx, test.NodeID, test.Description)
			if test.ExpError {
				assert.Error(t, err)
				return
			}
			if !assert.NoError(t, err) {
				return
			}
			col, err := d.db.Collection(ctx, COLLECTION_NODES)
			assert.NoError(t, err)
			node := Node{}
			meta, err := col.ReadDocument(ctx, test.NodeID, &node)
			assert.NoError(t, err, meta)
			assert.Equal(t, node.Description, test.ExpDescription)
		})
	}
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
			ExpSchema:        &SchemaOptionsNode,
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
			ExpSchemaChanged: false,
			ExpSchema:        &SchemaOptionsNode,
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
			ExpSchemaChanged: true,
			ExpSchema:        &SchemaOptionsNode,
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
				props.Schema.Rule.(map[string]interface{})["required"] = append(SchemaRequiredPropertiesNodes, "newkey")
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
			col, err := db.db.Collection(ctx, COLLECTION_NODES)
			assert.NoError(err)
			props, err := col.Properties(ctx)
			assert.NoError(err)
			if test.ExpSchema != nil {
				assert.Equal(test.ExpSchema, props.Schema)
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
		ExpError     bool
	}{
		{
			Name: "single translation: language 'en'",
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
			_, db, err := dbTestSetupCleanup(t)
			if err != nil {
				return
			}
			ctx := context.Background()
			assert := assert.New(t)
			id, err := db.CreateNode(ctx, &model.Text{
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
			assert.NotNil(n)
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
			_, db, err := dbTestSetupCleanup(t)
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
			_, db, err := dbTestSetupCleanup(t)
			if err != nil {
				return
			}
			if len(test.ExistingUsers) >= 1 {
				if err := setupDBWithUsers(t, db, test.ExistingUsers); err != nil {
					return
				}
			}
			ctx := context.Background()
			res, err := db.createUser(ctx, test.User, test.Password)
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
				assert.Empty(res.NewUserID, "there should not be a user ID, if creation fails")
				assert.Empty(users, "there should be no users in DB")
				return
			}
			assert.NotEmpty(res.NewUserID)
			if !assert.Len(users, 1, "one user should be created in DB") {
				return
			}
			assert.Equal(users[0].Username, test.User.Username)
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
			_, db, err := dbTestSetupCleanup(t)
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
				assert.Empty(res.NewUserID, "there should not be a user ID, if creation fails")
				assert.Empty(users, "there should be no users in DB")
				return
			}
			assert.NotEmpty(res.NewUserID)
			if !assert.Len(users, 1, "one user should be created in DB") {
				return
			}
			assert.Equal(users[0].Username, test.UserName)
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

func setupDBWithUsers(t *testing.T, db *ArangoDB, users []User) error {
	ctx := context.Background()
	col, err := db.db.Collection(ctx, COLLECTION_USERS)
	if !assert.NoError(t, err) {
		return err
	}
	for _, user := range users {
		meta, err := col.CreateDocument(ctx, user)
		if !assert.NoError(t, err, meta) {
			return err
		}
	}
	return nil
}

func TestArangoDB_Login(t *testing.T) {
	for _, test := range []struct {
		TestName              string
		EMail, Password       string
		ExpectError           bool
		ExpectLoginSuccess    bool
		ExpectErrorMessage    string
		Result                model.LoginResult
		ExistingUsers         []User
		TokenAmountAfterLogin int
	}{
		{
			TestName:           "user does not exist",
			EMail:              "abc@def.com",
			Password:           "1234567890",
			ExpectError:        false,
			ExpectLoginSuccess: false,
			ExpectErrorMessage: "User does not exist",
		},
		{
			TestName:           "password missmatch",
			EMail:              "abc@def.com",
			Password:           "1234567890",
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
			TestName:           "successful login",
			EMail:              "abc@def.com",
			Password:           "1234567890",
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
			_, db, err := dbTestSetupCleanup(t)
			if err != nil {
				return
			}
			if len(test.ExistingUsers) >= 1 {
				if err := setupDBWithUsers(t, db, test.ExistingUsers); err != nil {
					return
				}
			}
			ctx := context.Background()
			res, err := db.Login(ctx, test.EMail, test.Password)
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
			_, err = uuid.Parse(res.Token)
			assert.NoError(err)
			users, err := QueryReadAll[User](ctx, db, `FOR u in users FILTER u.email == @name RETURN u`, map[string]interface{}{
				"name": test.EMail,
			})
			assert.NoError(err)
			assert.Len(users, 1)
			user := users[0]
			if !assert.Len(user.Tokens, test.TokenAmountAfterLogin) {
				return
			}
			assert.Equal(user.Tokens[test.TokenAmountAfterLogin-1].Token, res.Token)
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
						{Token: "TOKEN"},
					},
				},
			},
			MakeCtxFn: func(ctx context.Context) context.Context {
				return middleware.CtxNewWithAuthentication(ctx, "TOKEN")
			},
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
				return middleware.CtxNewWithAuthentication(ctx, "BBBBBB")
			},
			ExpectError: true,
		},
	} {
		t.Run(test.TestName, func(t *testing.T) {
			_, db, err := dbTestSetupCleanup(t)
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
			_, db, err := dbTestSetupCleanup(t)
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

//func TestArangoDB_DeleteAccount(t *testing.T) {
//	for _, test := range []struct {
//		Name string
//	}{
//		{},
//	}{
//		t.Run(test.Name, func(t*testing.T) {
//		})
//	}
//}
