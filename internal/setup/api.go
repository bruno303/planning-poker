package setup

import (
	"github.com/gorilla/mux"
)

func ConfigureAPIs(r *mux.Router, container *Container) {
	for _, api := range container.API.APIs {
		route := r.Handle(api.Endpoint(), api.Handle())

		methods := api.Methods()
		if len(methods) > 0 {
			route.Methods(methods...)
		}
	}
}
