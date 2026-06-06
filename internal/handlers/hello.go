package handlers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type HelloHandler struct {
	logger *zap.Logger
}

func NewHelloHandler(logger *zap.Logger) *HelloHandler {
	return &HelloHandler{logger: logger}
}

// Hello handles GET /hello
// @Summary      Hello World Endpoint
// @Description  Simple endpoint that returns a Hello World message in JSON format.
// @Tags         system
// @Produce      json
// @Success      200      {object}  map[string]string "Hello World message"
// @Router       /hello [get]
func (h *HelloHandler) Hello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Hello World!"}); err != nil {
		h.logger.Error("failed to encode hello response", zap.Error(err))
	}
}
