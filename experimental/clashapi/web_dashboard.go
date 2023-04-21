//go:build with_clash_ui

package clashapi

import (
	"embed"
	"github.com/go-chi/chi/v5"
	"io/fs"
	"net/http"
	"path"
)

//go:embed web_dashboard
var webDir embed.FS

type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

func initWebDir(chiRouter *chi.Mux) {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join("web_dashboard", name)
		file, err := webDir.Open(assetPath)
		if err != nil {
			return nil, err
		}
		return file, err
	})

	chiRouter.Group(func(r chi.Router) {
		r.Get("/ui", http.RedirectHandler("/ui/", http.StatusTemporaryRedirect).ServeHTTP)
		r.Get("/ui/*", http.StripPrefix("/ui", http.FileServer(http.FS(handler))).ServeHTTP)
	})
}
