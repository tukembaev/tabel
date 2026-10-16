package httpapi

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed swagger-ui/*
var swaggerUIFS embed.FS

//go:embed openapi.yaml
var openAPISpec []byte

func mountSwagger(r chi.Router) {
	r.Get("/swagger/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(openAPISpec)
	})
	r.Get("/swagger", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/", http.StatusFound)
	})

	sub, err := fs.Sub(swaggerUIFS, "swagger-ui")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	r.Handle("/swagger/*", http.StripPrefix("/swagger/", fileServer))
}
