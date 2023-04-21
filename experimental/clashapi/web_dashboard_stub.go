//go:build !with_clash_ui

package clashapi

import (
	"github.com/go-chi/chi/v5"
)

func initWebDir(chiRouter *chi.Mux) {
}
