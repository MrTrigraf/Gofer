package http

import (
	"encoding/json"
	"net/http"

	"github.com/gofer/internal/usecase/user"
	"github.com/gofer/pkg/httputil"
)

type UserHandler struct {
	userUC *user.UserUseCase
}

func NewUserHandler(userUC *user.UserUseCase) *UserHandler {
	return &UserHandler{userUC: userUC}
}

func (h *UserHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		httputil.WriteError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	users, err := h.userUC.SearchUsers(r.Context(), query)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
