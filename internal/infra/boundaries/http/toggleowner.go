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
	ToggleOwnerAPI struct {
		adminToggleOwner    usecase.UseCase[usecase.AdminToggleOwnerCommand]
		adminAuthMiddleware middleware.AdminMiddleware
		logger              log.Logger
	}
)

var _ API = (*ToggleOwnerAPI)(nil)

// @Summary Toggle owner status
// @Description Toggles the owner status for a client in a room (admin only)
// @Tags admin
// @Produce json
// @Param roomID path string true "Room ID"
// @Param clientID path string true "Client ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/rooms/{roomID}/client/{clientID}/owner [post]
func NewToggleOwnerAPI(
	adminToggleOwner usecase.UseCase[usecase.AdminToggleOwnerCommand],
	adminAuthMiddleware middleware.AdminMiddleware,
) ToggleOwnerAPI {
	return ToggleOwnerAPI{
		adminToggleOwner:    adminToggleOwner,
		adminAuthMiddleware: adminAuthMiddleware,
		logger:              log.NewLogger("toggleownerapi"),
	}
}

func (api ToggleOwnerAPI) Endpoint() string {
	return "/admin/rooms/{roomID}/client/{clientID}/owner"
}

func (api ToggleOwnerAPI) Methods() []string {
	return []string{"POST", "OPTIONS"}
}

func (api ToggleOwnerAPI) Handle() http.Handler {
	return api.adminAuthMiddleware.Handle(api.execute())
}

func (api ToggleOwnerAPI) execute() http.Handler {
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

		err := api.adminToggleOwner.Execute(ctx, usecase.AdminToggleOwnerCommand{
			RoomID:         roomID,
			TargetClientID: clientID,
		})
		if errors.Is(err, domain.ErrLastOwner) {
			SendJsonErrorMsg(w, http.StatusBadRequest, "Cannot remove the last owner")
			return
		}
		if errors.Is(err, domain.ErrRoomNotFound) {
			SendJsonErrorMsg(w, http.StatusNotFound, "Room not found")
			return
		}
		if errors.Is(err, domain.ErrClientNotFound) {
			SendJsonErrorMsg(w, http.StatusNotFound, "Client not found")
			return
		}
		if err != nil {
			api.logger.Error(ctx, "Failed to toggle owner", err)
			SendJsonErrorMsg(w, http.StatusInternalServerError, "Failed to toggle owner")
			return
		}

		SendJsonResponse(w, http.StatusOK, map[string]string{"status": "owner-toggled"})
	})
}
