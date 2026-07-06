package http

import (
	"embed"
	"net/http"

	"github.com/gorilla/mux"
)

//go:embed swagger/swagger.yaml swagger/swagger.json
var swaggerFS embed.FS

type SwaggerAPI struct{}

var _ API = (*SwaggerAPI)(nil)

func NewSwaggerAPI() SwaggerAPI {
	return SwaggerAPI{}
}

func (s SwaggerAPI) Endpoint() string {
	return "/swagger/{rest:.*}"
}

func (s SwaggerAPI) Methods() []string {
	return []string{"GET"}
}

func (s SwaggerAPI) Handle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest := mux.Vars(r)["rest"]
		switch rest {
		case "", "/":
			serveSwaggerUI(w, r)
		case "swagger.yaml":
			data, err := swaggerFS.ReadFile("swagger/swagger.yaml")
			if err != nil {
				http.Error(w, "Failed to load OpenAPI spec", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/x-yaml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		case "swagger.json":
			data, err := swaggerFS.ReadFile("swagger/swagger.json")
			if err != nil {
				http.Error(w, "Failed to load OpenAPI spec", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		default:
			http.NotFound(w, r)
		}
	})
}

func serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Planning Poker API - Swagger UI</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: "/swagger/swagger.yaml",
            dom_id: "#swagger-ui",
        });
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
