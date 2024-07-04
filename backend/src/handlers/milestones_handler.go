package handlers

import (
	"UnlockEdv2/src/database"
	"UnlockEdv2/src/models"
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func (srv *Server) registerMilestonesRoutes() {
	srv.Mux.Handle("GET /api/milestones", srv.applyMiddleware(http.HandlerFunc(srv.HandleIndexMilestones)))
	srv.Mux.Handle("POST /api/milestones", srv.ApplyAdminMiddleware(http.HandlerFunc(srv.HandleCreateMilestone)))
	srv.Mux.Handle("DELETE /api/milestones", srv.ApplyAdminMiddleware(http.HandlerFunc(srv.HandleDeleteMilestone)))
	srv.Mux.Handle("PATCH /api/milestones/{id}", srv.ApplyAdminMiddleware(http.HandlerFunc(srv.HandleUpdateMilestone)))
}

func (srv *Server) HandleIndexMilestones(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	orderBy := r.URL.Query().Get("order_by")
	page, perPage := srv.GetPaginationInfo(r)
	var milestones []database.MilestoneResponse
	err := error(nil)
	total := int64(0)
	if !srv.UserIsAdmin(r) {
		userId := r.Context().Value(ClaimsKey).(*Claims).UserID
		total, milestones, err = srv.Db.GetMilestonesForUser(page, perPage, userId)
		if err != nil {
			log.Debug("IndexMilestones Database Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		total, milestones, err = srv.Db.GetMilestones(page, perPage, search, orderBy)
		if err != nil {
			log.Debug("IndexMilestones Database Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	last := srv.CalculateLast(total, perPage)
	paginationData := models.PaginationMeta{
		PerPage:     perPage,
		LastPage:    int(last),
		CurrentPage: page,
		Total:       total,
	}
	response := models.PaginatedResource[database.MilestoneResponse]{
		Meta: paginationData,
		Data: milestones,
	}
	srv.WriteResponse(w, http.StatusOK, response)
}

func (srv *Server) HandleCreateMilestone(w http.ResponseWriter, r *http.Request) {
	miles := &models.Milestone{}
	if err := json.NewDecoder(r.Body).Decode(miles); err != nil {
		srv.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()
	if _, err := srv.Db.CreateMilestone(miles); err != nil {
		srv.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	srv.WriteResponse(w, http.StatusCreated, miles)
}

func (srv *Server) HandleDeleteMilestone(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		srv.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := srv.Db.DeleteMilestone(id); err != nil {
		srv.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (srv *Server) HandleUpdateMilestone(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Info("No ID provided in URL, checking request body json")
	}
	miles := &models.Milestone{}
	if err := json.NewDecoder(r.Body).Decode(miles); err != nil {
		srv.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()
	msId := 0
	if id != 0 {
		msId = int(id)
	} else if miles.ID != 0 {
		msId = int(miles.ID)
	} else {
		srv.ErrorResponse(w, http.StatusBadRequest, "No ID provided in URL or request body")
	}
	toUpdate, err := srv.Db.GetMilestoneByID(msId)
	if err != nil {
		srv.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	models.UpdateStruct(&toUpdate, &miles)
	if _, err := srv.Db.UpdateMilestone(toUpdate); err != nil {
		srv.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	srv.WriteResponse(w, http.StatusOK, toUpdate)
}
