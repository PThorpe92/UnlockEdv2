package handlers

import (
	"backend/cmd/models"
	"encoding/json"
	"net/http"
	"strconv"
)

func (srv *Server) RegisterUserRoutes() {
	srv.Mux.Handle("GET /api/users", srv.ApplyMiddleware(http.HandlerFunc(srv.IndexUsers)))
	srv.Mux.Handle("GET /api/users/{id}", srv.ApplyMiddleware(http.HandlerFunc(srv.GetUserByID)))
	srv.Mux.Handle("POST /api/users", srv.ApplyMiddleware(http.HandlerFunc(srv.CreateUser)))
	srv.Mux.Handle("DELETE /api/users/{id}", srv.ApplyMiddleware(http.HandlerFunc(srv.DeleteUser)))
	srv.Mux.Handle("PATCH /api/users/{id}", srv.ApplyMiddleware(http.HandlerFunc(srv.UpdateUser)))
}

/**
* GET: /api/v1/users
**/
func (srv *Server) IndexUsers(w http.ResponseWriter, r *http.Request) {
	srv.Logger.Println("HIT: Index users")
	page, perPage := srv.GetPaginationInfo(r)
	total, users, err := srv.Db.GetCurrentUsers(page, perPage)
	if err != nil {
		srv.Logger.Printf("IndexUsers Database Error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	last := srv.CalculateLast(total, perPage)
	paginationData := models.PaginationMeta{
		PerPage:     perPage,
		LastPage:    int(last),
		CurrentPage: page,
		Total:       total,
	}
	srv.Logger.Printf("IndexUsers: %v", users)
	response := models.PaginatedResource[models.User]{
		Data: users,
		Meta: paginationData,
	}
	if err = srv.WriteResponse(w, http.StatusOK, response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/**
* GET: /api/v1/users/{id}
**/
func (srv *Server) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		srv.Logger.Printf("GET User handler Error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	response := models.Resource[models.User]{}
	user, err := srv.Db.GetUserByID(id)
	if err != nil {
		srv.Logger.Printf("Error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response.Data = append(response.Data, user)
	bytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

/**
* POST: /api/v1/users
**/
func (srv *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
	user := models.User{}
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		srv.Logger.Printf("POST User handler Error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	temp, err := srv.Db.CreateUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := srv.WriteResponse(w, http.StatusCreated, map[string]string{"password": temp}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/**
* DELETE: /api/v1/users/{id}
 */
func (srv *Server) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		srv.Logger.Printf("DELETE User handler Error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if err = srv.Db.DeleteUser(id); err != nil {
		srv.ErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
	w.WriteHeader(http.StatusNoContent)
}

/**
* PATCH: /api/v1/users/{id}
**/
func (srv *Server) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		srv.Logger.Printf("UPDATE User handler Error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	user := models.User{}
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user.ID = id
	updatedUser, err := srv.Db.UpdateUser(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := models.Resource[models.User]{}
	response.Data = append(response.Data, updatedUser)
	if err := srv.WriteResponse(w, http.StatusOK, response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}