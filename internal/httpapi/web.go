package httpapi

import (
	_ "embed"
	"net/http"
)

//go:embed web/index.html
var indexHTML []byte

// webIndex serves the operator page for the normalized timeline and conflict evidence.
func (h *Handler) webIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(indexHTML)
}
