package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	projectApp "github.com/safarislava/typstlab-server/internal/application/project"
	projectHttp "github.com/safarislava/typstlab-server/internal/infrastructure/http"
	projectDb "github.com/safarislava/typstlab-server/internal/infrastructure/persistence"
)

func main() {
	r := setupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(":"+port, r))
}

func setupRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	repo := projectDb.NewMemoryProjectRepository()
	service := projectApp.NewService(repo)
	handler := projectHttp.NewProjectHandler(service)

	r.Post("/projects", handler.Create)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return r
}
