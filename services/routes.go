package services

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

//Routes return all services admin routes
func Routes() *chi.Mux {
	r := chi.NewRouter()

	// v1/api/admin/services
	r.Route("/", func(r chi.Router) {
		r.Get("/", getAllServices)
		r.Get("/reload", reloadServices)
	})

	return r
}

func getAllServices(w http.ResponseWriter, r *http.Request) {
	render.Status(r, 200)
	render.JSON(w, r, srv.List)
}

func reloadServices(w http.ResponseWriter, r *http.Request) {
	srv.Load(true)
	render.Status(r, 200)
	render.JSON(w, r, srv.List)
}
