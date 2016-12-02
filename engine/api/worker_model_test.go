package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"encoding/json"

	"fmt"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func Test_DeleteAllWorkerModel(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)

	//Loading all models
	dbmap := database.DBMap(db)
	models, err := worker.LoadWorkerModels(dbmap)
	if err != nil {
		t.Fatalf("Error getting models : %s", err)
	}

	//Delete all of them
	for _, m := range models {
		if err := worker.DeleteWorkerModel(dbmap, m.ID); err != nil {
			t.Fatalf("Error deleting model : %s", err)
		}
	}

}

func Test_addWorkerModelAsAdmin(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addWorkerModelAsAdmin"}
	router.init()

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelWithWrongRequest(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addWorkerModelAsAdmin"}
	router.init()

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	//Type is mandatory
	model := sdk.Model{
		Name:    "Test1",
		Image:   "buildpack-deps:jessie",
		GroupID: g.ID,
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//Name is mandatory
	model = sdk.Model{
		GroupID: g.ID,
		Type:    sdk.Docker,
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	req = testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//GroupID is mandatory
	model = sdk.Model{
		Name:  "Test1",
		Type:  sdk.Docker,
		Image: "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	req = testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//SendBadRequest

	//Prepare request
	req = testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, "blabla")

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelAsAGroupMember(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addWorkerModelAsAGroupMember"}
	router.init()

	//Create group
	g := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	//Create user
	u, pass, err := testwithdb.InsertLambaUser(t, db, g)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelAsAGroupAdmin(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addWorkerModelAsAGroupMember"}
	router.init()

	//Create group
	g := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	//Create user
	u, pass, err := testwithdb.InsertLambaUser(t, db, g)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	if err := group.SetUserGroupAdmin(db, g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelAsAWrongGroupMember(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addWorkerModelAsAGroupMember"}
	router.init()

	//Create group
	g := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	//Create group
	g1 := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	if err := group.InsertGroup(db, g1); err != nil {
		t.Fatal(err)
	}

	//Create user
	u, pass, err := testwithdb.InsertLambaUser(t, db, g)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	if err := group.SetUserGroupAdmin(db, g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g1.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_updateWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_updateWorkerModel"}
	router.init()

	//Create group
	g := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	//Create user
	u, pass, err := testwithdb.InsertLambaUser(t, db, g)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	if err := group.SetUserGroupAdmin(db, g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	json.Unmarshal(w.Body.Bytes(), &model)

	model2 := sdk.Model{
		Name: "Test1bis",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
			{
				Name:  "capa2",
				Type:  sdk.BinaryRequirement,
				Value: "2",
			},
		},
	}

	//Prepare request
	vars := map[string]string{
		"permModelID": fmt.Sprintf("%d", model.ID),
	}
	uri = router.getRoute("PUT", updateWorkerModel, vars)
	if uri == "" {
		t.FailNow()
	}
	req = testwithdb.NewAuthentifiedRequest(t, u, pass, "PUT", uri, model2)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

}

func Test_deleteWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_deleteWorkerModel"}
	router.init()

	//Create group
	g := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	//Create user
	u, pass, err := testwithdb.InsertLambaUser(t, db, g)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	if err := group.SetUserGroupAdmin(db, g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	json.Unmarshal(w.Body.Bytes(), &model)

	//Prepare request
	vars := map[string]string{
		"permModelID": fmt.Sprintf("%d", model.ID),
	}
	uri = router.getRoute("DELETE", deleteWorkerModel, vars)
	if uri == "" {
		t.FailNow()
	}
	req = testwithdb.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

}

func Test_getWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addWorkerModelAsAdmin"}
	router.init()

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//Prepare request
	uri = router.getRoute("GET", getWorkerModels, nil)
	if uri == "" {
		t.FailNow()
	}

	req = testwithdb.NewAuthentifiedRequest(t, u, pass, "GET", uri+"?name=Test1", nil)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

}

func Test_getWorkerModels(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_addWorkerModelAsAdmin"}
	router.init()

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	g, err := group.LoadGroup(db, "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.getRoute("POST", addWorkerModel, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//Prepare request
	uri = router.getRoute("GET", getWorkerModels, nil)
	if uri == "" {
		t.FailNow()
	}
	req = testwithdb.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	results := []sdk.Model{}
	json.Unmarshal(w.Body.Bytes(), &results)

	assert.Equal(t, 1, len(results))
	assert.Equal(t, "Test1", results[0].Name)
	assert.Equal(t, 1, len(results[0].Capabilities))
	assert.Equal(t, u.Fullname, results[0].CreatedBy.Fullname)
}

func Test_addWorkerModelCapa(t *testing.T) {

}

func Test_getWorkerModelTypes(t *testing.T) {

}

func Test_getWorkerModelCapaTypes(t *testing.T) {

}

func Test_updateWorkerModelCapa(t *testing.T) {

}

func Test_deleteWorkerModelCapa(t *testing.T) {

}
