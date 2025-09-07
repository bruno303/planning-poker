package infra

import (
	"net/http"

	"github.com/gorilla/mux"
)

func ConfigureInfraAPI(mux *mux.Router) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}
