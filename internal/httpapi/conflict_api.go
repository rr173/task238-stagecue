package httpapi

import (
	"net/http"

	"task238-stagecue/internal/model"
)

// listConflicts 处理 GET /api/rehearsals/:id/conflicts。
func (h *Handler) listConflicts(w http.ResponseWriter, r *http.Request, id string) {
	cf, err := h.svc.Store.Conflicts().ListByRehearsal(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cf)
}

// getConflict 处理 GET /api/conflicts/:id。
func (h *Handler) getConflict(w http.ResponseWriter, r *http.Request, id string) {
	cf, err := h.svc.Store.Conflicts().Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cf)
}

// waiverConflict 处理 POST /api/conflicts/:id/waiver：登记豁免并解决冲突。
func (h *Handler) waiverConflict(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Reason   string `json:"reason"`
		Evidence string `json:"evidence"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrMissingWaiver)
		return
	}
	wv, err := h.svc.Waiver.Register(id, body.Reason, body.Evidence)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, wv)
}
