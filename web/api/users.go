/*
 Copyright 2018 Padduck, LLC
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 	http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package api

import (
	"github.com/gin-gonic/gin"
	builder "github.com/pufferpanel/apufferi/response"
	"github.com/pufferpanel/pufferpanel"
	"github.com/pufferpanel/pufferpanel/database"
	"github.com/pufferpanel/pufferpanel/models"
	"github.com/pufferpanel/pufferpanel/services"
	"github.com/pufferpanel/pufferpanel/web/handlers"
	"net/http"
)

func registerUsers(g *gin.RouterGroup) {
	//if you can log in, you can see and edit yourself
	g.Handle("GET", "", handlers.OAuth2(pufferpanel.ScopeLogin, false), getSelf)
	g.Handle("PUT", "", handlers.OAuth2(pufferpanel.ScopeLogin, false), updateSelf)
	g.Handle("POST", "", handlers.OAuth2(pufferpanel.ScopeViewUsers, false), searchUsers)
	g.Handle("OPTIONS", "", pufferpanel.CreateOptions("GET", "PUT", "POST"))

	g.Handle("PUT", "/:username", handlers.OAuth2(pufferpanel.ScopeEditUsers, false), createUser)
	g.Handle("GET", "/:username", handlers.OAuth2(pufferpanel.ScopeViewUsers, false), getUser)
	g.Handle("POST", "/:username", handlers.OAuth2(pufferpanel.ScopeEditUsers, false), updateUser)
	g.Handle("DELETE", "/:username", handlers.OAuth2(pufferpanel.ScopeEditUsers, false), deleteUser)
	g.Handle("OPTIONS", "/:username", pufferpanel.CreateOptions("PUT", "GET", "POST", "DELETE"))
}

func searchUsers(c *gin.Context) {
	var err error
	response := builder.From(c)

	search := newUserSearch()
	err = c.BindJSON(search)
	if pufferpanel.HandleError(response, err) {
		return
	}
	if search.PageLimit <= 0 {
		response.Fail().Status(http.StatusBadRequest).Message("page size must be a positive number")
		return
	}

	if search.PageLimit > MaxPageSize {
		search.PageLimit = MaxPageSize
	}

	if search.Page <= 0 {
		response.Fail().Status(http.StatusBadRequest).Message("page must be a positive number")
		return
	}

	db, err := database.GetConnection()
	if pufferpanel.HandleError(response, err) {
		return
	}

	us := &services.User{DB: db}

	var results *models.Users
	var total uint
	if results, total, err = us.Search(search.Username, search.Email, uint(search.PageLimit), uint(search.Page)); pufferpanel.HandleError(response, err) {
		return
	}

	response.PageInfo(uint(search.Page), uint(search.PageLimit), MaxPageSize, total).Data(models.FromUsers(results))
}

func createUser(c *gin.Context) {
	var err error
	response := builder.Respond(c)

	db, err := database.GetConnection()
	if pufferpanel.HandleError(response, err) {
		return
	}

	us := &services.User{DB: db}

	var viewModel models.UserView
	if err = c.BindJSON(&viewModel); pufferpanel.HandleError(response, err) {
		return
	}
	viewModel.Username = c.Param("username")

	if err = viewModel.Valid(false); pufferpanel.HandleError(response, err) {
		return
	}

	if viewModel.Password == "" {
		pufferpanel.HandleError(response, pufferpanel.ErrFieldRequired("password"))
		return
	}

	user := &models.User{}
	viewModel.CopyToModel(user)

	if err = us.Create(user); pufferpanel.HandleError(response, err) {
		return
	}

	response.Data(models.FromUser(user))
}

func getUser(c *gin.Context) {
	response := builder.Respond(c)

	db, err := database.GetConnection()
	if pufferpanel.HandleError(response, err) {
		return
	}

	us := &services.User{DB: db}

	username := c.Param("username")

	user, exists, err := us.Get(username)
	if pufferpanel.HandleError(response, err) {
		return
	} else if !exists {
		response.Fail().Status(http.StatusNotFound).Message("no user with username")
		return
	}

	response.Data(models.FromUser(user))
}

func getSelf(c *gin.Context) {
	response := builder.Respond(c)

	t, exist := c.Get("user")
	user, ok := t.(*models.User)

	if !exist || !ok {
		response.Fail().Status(http.StatusNotFound).Message("no user with username")
		return
	}

	response.Data(models.FromUser(user))
}

func updateSelf(c *gin.Context) {
	response := builder.Respond(c)

	db, err := database.GetConnection()
	if pufferpanel.HandleError(response, err) {
		return
	}

	us := &services.User{DB: db}

	t, exist := c.Get("user")
	user, ok := t.(*models.User)

	if !exist || !ok {
		response.Fail().Status(http.StatusNotFound).Message("no user with username")
		return
	}

	var viewModel models.UserView
	if err = c.BindJSON(&viewModel); pufferpanel.HandleError(response, err) {
		return
	}

	if err = viewModel.Valid(true); pufferpanel.HandleError(response, err) {
		return
	}

	if viewModel.Password == "" {
		return
	}

	if us.IsValidCredentials(user, viewModel.Password) {
		pufferpanel.HandleError(response, pufferpanel.ErrInvalidCredentials)
		return
	}

	viewModel.CopyToModel(user)

	if err = us.Update(user); pufferpanel.HandleError(response, err) {
		return
	}

	response.Data(models.FromUser(user))
}

func updateUser(c *gin.Context) {
	response := builder.Respond(c)

	db, err := database.GetConnection()
	if pufferpanel.HandleError(response, err) {
		return
	}

	us := &services.User{DB: db}

	username := c.Param("username")

	var viewModel models.UserView
	if err = c.BindJSON(&viewModel); pufferpanel.HandleError(response, err) {
		return
	}

	if err = viewModel.Valid(true); pufferpanel.HandleError(response, err) {
		return
	}

	user, exists, err := us.Get(username)
	if pufferpanel.HandleError(response, err) {
		return
	} else if !exists {
		response.Fail().Status(http.StatusNotFound).Message("no user with username")
		return
	}

	viewModel.CopyToModel(user)

	if err = us.Update(user); pufferpanel.HandleError(response, err) {
		return
	}

	response.Data(models.FromUser(user))
}

func deleteUser(c *gin.Context) {
	response := builder.Respond(c)

	db, err := database.GetConnection()
	if pufferpanel.HandleError(response, err) {
		return
	}

	us := &services.User{DB: db}

	username := c.Param("username")

	user, exists, err := us.Get(username)
	if pufferpanel.HandleError(response, err) {
		return
	} else if !exists {
		response.Fail().Status(http.StatusNotFound).Message("no user with username")
		return
	}

	if err = us.Delete(user.Username); pufferpanel.HandleError(response, err) {
		return
	}

	response.Data(models.FromUser(user))
}

type UserSearch struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	PageLimit int    `json:"limit"`
	Page      int    `json:"page"`
}

func newUserSearch() *UserSearch {
	return &UserSearch{
		Username:  "*",
		Email:     "*",
		PageLimit: DefaultPageSize,
		Page:      1,
	}
}