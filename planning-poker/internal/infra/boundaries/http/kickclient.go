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
	KickClientAPI struct {
		adminKickClient     usecase.UseCase[usecase.AdminKickClientCommand]
		adminAuthMiddleware middleware.AdminMiddleware
		logger              log.Logger
	}
)

var _ API = (*KickClientAPI)(nil)

// @Summary Kick a client
// @Description Kicks a client from a room (admin only)
// @Tags admin
// @Produce json
// @Param roomID path string true "Room ID"
// @Param clientID path string true "Client ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/rooms/{roomID}/client/{clientID}/kick [post]
func NewKickClientAPI(
	adminKickClient usecase.UseCase[usecase.AdminKickClientCommand],
	adminAuthMiddleware middleware.AdminMiddleware,
) KickClientAPI {
	return KickClientAPI{
		adminKickClient:     adminKickClient,
		adminAuthMiddleware: adminAuthMiddleware,
		logger:              log.NewLogger("kickclientapi"),
	}
}

func (api KickClientAPI) Endpoint() string {
	return "/admin/rooms/{roomID}/client/{clientID}/kick"
}

func (api KickClientAPI) Methods() []string {
	return []string{"POST", "OPTIONS"}
}

func (api KickClientAPI) Handle() http.Handler {
	return api.adminAuthMiddleware.Handle(api.execute())
}

func (api KickClientAPI) execute() http.Handler {
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

		err := api.adminKickClient.Execute(ctx, usecase.AdminKickClientCommand{
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
			api.logger.Error(ctx, "Failed to kick client from room", err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to kick client")
			return
		}

		SendJsonResponse(w, http.StatusOK, map[string]string{"status": "kicked"})
	})
}
