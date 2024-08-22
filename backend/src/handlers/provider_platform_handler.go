package handlers

import (
	"UnlockEdv2/src/models"
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func (srv *Server) registerProviderPlatformRoutes() {
	srv.Mux.HandleFunc("GET /api/provider-platforms", srv.ApplyAdminMiddleware(srv.HandleIndexProviders))
	srv.Mux.HandleFunc("GET /api/provider-platforms/{id}", srv.ApplyAdminMiddleware(srv.HandleShowProvider))
	srv.Mux.HandleFunc("POST /api/provider-platforms", srv.ApplyAdminMiddleware(srv.HandleCreateProvider))
	srv.Mux.HandleFunc("PATCH /api/provider-platforms/{id}", srv.ApplyAdminMiddleware(srv.HandleUpdateProvider))
	srv.Mux.HandleFunc("DELETE /api/provider-platforms/{id}", srv.ApplyAdminMiddleware(srv.HandleDeleteProvider))
}

func (srv *Server) HandleIndexProviders(w http.ResponseWriter, r *http.Request) {
	page, perPage := srv.GetPaginationInfo(r)
	total, platforms, err := srv.Db.GetAllProviderPlatforms(page, perPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	paginationData := models.NewPaginationInfo(page, perPage, total)
	response := models.PaginatedResource[models.ProviderPlatform]{
		Meta: paginationData,
	}
	only := r.URL.Query().Get("only")
	if only == "oidc_enabled" {
		for _, platform := range platforms {
			if platform.OidcID != 0 {
				response.Data = append(response.Data, platform)
			}
		}
	} else {
		response.Data = platforms
	}
	log.Info("Found "+strconv.Itoa(int(total)), " provider platforms")
	srv.WriteResponse(w, http.StatusOK, response)
}

func (srv *Server) HandleShowProvider(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Error("GET Provider handler Error: ", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	platform, err := srv.Db.GetProviderPlatformByID(id)
	if err != nil {
		log.Error("Error: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := models.Resource[models.ProviderPlatform]{
		Data:    []models.ProviderPlatform{*platform},
		Message: "Provider platform found",
	}
	srv.WriteResponse(w, http.StatusOK, response)
}

func (srv *Server) HandleCreateProvider(w http.ResponseWriter, r *http.Request) {
	var platform models.ProviderPlatform
	err := json.NewDecoder(r.Body).Decode(&platform)
	if err != nil {
		log.Error("Error decoding request body: ", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	newProv, err := srv.Db.CreateProviderPlatform(&platform)
	if err != nil {
		log.Error("Error creating provider platform: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := models.Resource[models.ProviderPlatform]{
		Data:    make([]models.ProviderPlatform, 0),
		Message: "Provider platform created successfully",
	}
	response.Data = append(response.Data, *newProv)
	srv.WriteResponse(w, http.StatusOK, response)
}

func (srv *Server) HandleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Error("PATCH Provider handler Error:", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var platform models.ProviderPlatform
	err = json.NewDecoder(r.Body).Decode(&platform)
	if err != nil {
		log.Error("Error decoding request body: ", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	updated, err := srv.Db.UpdateProviderPlatform(&platform, uint(id))
	if err != nil {
		log.Error("Error updating provider platform: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := models.Resource[models.ProviderPlatform]{
		Data: make([]models.ProviderPlatform, 0),
	}
	response.Data = append(response.Data, *updated)
	srv.WriteResponse(w, http.StatusOK, response)
}

func (srv *Server) HandleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Error("DELETE Provider handler Error: ", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err = srv.Db.DeleteProviderPlatform(id); err != nil {
		log.Error("Error deleting provider platform: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
