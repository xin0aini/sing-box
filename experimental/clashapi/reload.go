//go:build !windows

package clashapi

import (
	"net/http"
	"os"
	"syscall"

	"github.com/go-chi/render"
)

func reload(server *Server) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			server.logger.Warn("config file reloading...")
			pid := os.Getpid()
			err := syscall.Kill(pid, syscall.SIGHUP)
			if err != nil {
				server.logger.Error("failed to reload: ", err)
			}
		}()
		render.NoContent(w, r)
	}
}
