package http

import (
	"errors"
	"net/http"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/boundaries/http/middleware"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/gorilla/mux"
)

type (
	DisconnectClientAPI struct {
		adminRemoveClient   usecase.UseCase[usecase.AdminRemoveClientCommand]
		adminAuthMiddleware middleware.AdminMiddleware
		logger              log.Logger
	}
)

var _ API = (*DisconnectClientAPI)(nil)

func NewDisconnectClientAPI(
	adminRemoveClient usecase.UseCase[usecase.AdminRemoveClientCommand],
	adminAuthMiddleware middleware.AdminMiddleware,
) DisconnectClientAPI {
	return DisconnectClientAPI{
		adminRemoveClient:   adminRemoveClient,
		adminAuthMiddleware: adminAuthMiddleware,
		logger:              log.NewLogger("disconnectclientapi"),
	}
}

func (api DisconnectClientAPI) Endpoint() string {
	return "/admin/rooms/{roomID}/client/{clientID}"
}

func (api DisconnectClientAPI) Methods() []string {
	return []string{"DELETE", "OPTIONS"}
}

func (api DisconnectClientAPI) Handle() http.Handler {
	return api.adminAuthMiddleware.Handle(api.execute())
}

func (api DisconnectClientAPI) execute() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		roomID := mux.Vars(r)["roomID"]
		clientID := mux.Vars(r)["clientID"]

		if roomID == "" {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Room ID is required")
			return
		}
		if clientID == "" {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Client ID is required")
			return
		}

		err := api.adminRemoveClient.Execute(ctx, usecase.AdminRemoveClientCommand{
			RoomID:   roomID,
			ClientID: clientID,
		})
		if errors.Is(err, domain.ErrRoomNotFound) {
			SendJsonErrorMsg(w, http.StatusNotFound, "Room not found")
			return
		}
		if errors.Is(err, domain.ErrClientNotFound) {
			SendJsonErrorMsg(w, http.StatusNotFound, "Client not found")
			return
		}
		if err != nil {
			api.logger.Error(ctx, "Failed to disconnect client from room", err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to disconnect client")
			return
		}

		SendJsonResponse(w, http.StatusOK, map[string]string{"status": "disconnected"})
	})
}
