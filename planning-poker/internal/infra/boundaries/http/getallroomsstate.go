package http

import (
	"net/http"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/http/middleware"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	GetAllRoomsStateResponse struct {
		ID      string
		Clients []GetAllRoomsStateClient
	}
	GetAllRoomsStateClient struct {
		ID          string
		Name        string
		IsSpectator bool
		IsOwner     bool
	}
	GetAllRoomsStateAPI struct {
		hub                 domain.AdminHub
		adminAuthMiddleware middleware.AdminMiddleware
		logger              log.Logger
	}
)

var _ API = (*GetAllRoomsStateAPI)(nil)

func NewGetAllRoomsStateAPI(hub domain.AdminHub, apiKey string) GetAllRoomsStateAPI {
	return GetAllRoomsStateAPI{
		hub:                 hub,
		adminAuthMiddleware: middleware.NewAdminMiddleware(apiKey),
		logger:              log.NewLogger("getallroomsstateapi"),
	}
}

func (api GetAllRoomsStateAPI) Endpoint() string {
	return "/admin/rooms"
}

func (api GetAllRoomsStateAPI) Methods() []string {
	return []string{"GET"}
}

func (api GetAllRoomsStateAPI) Handle() http.Handler {
	return api.adminAuthMiddleware.Handle(api.execute())
}

func (api GetAllRoomsStateAPI) execute() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rooms := api.hub.GetRooms()

		SendJsonResponse(w, http.StatusOK, mapRooms(rooms))
	})
}

func mapRooms(rooms []*entity.Room) []GetAllRoomsStateResponse {
	res := make([]GetAllRoomsStateResponse, len(rooms))
	for i, room := range rooms {
		r := GetAllRoomsStateResponse{
			ID:      room.ID,
			Clients: mapClients(room.Clients),
		}
		res[i] = r
	}

	return res
}

func mapClients(clients entity.ClientCollection) []GetAllRoomsStateClient {
	res := make([]GetAllRoomsStateClient, clients.Count())

	for i, client := range clients.Values() {
		r := GetAllRoomsStateClient{
			ID:          client.ID,
			Name:        client.Name,
			IsSpectator: client.IsSpectator,
			IsOwner:     client.IsOwner,
		}
		res[i] = r
	}

	return res
}
