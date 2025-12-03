package http

import "net/http"

type API interface {
	Endpoint() string
	Methods() []string
	Handle() http.Handler
}
