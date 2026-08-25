package httpapi

import (
	"net/http"

	"task238-stagecue/internal/model"
)

// listWaivers 处理 GET /api/waivers?rehearsal_id=。
func (h *Handler) listWaivers(w http.ResponseWriter, r *http.Request) {
	rid := r.URL.Query().Get("rehearsal_id")
	if rid == "" {
		writeError(w, model.ErrUnknownRehearsal)
		return
	}
	wvs, err := h.svc.Waiver.List(rid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wvs)
}

// waiverResolution 处理 GET /api/conflicts/:id/resolution。
func (h *Handler) waiverResolution(w http.ResponseWriter, r *http.Request, id string) {
	resolved, reason, err := h.svc.Waiver.ConflictResolution(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"conflict_id": id,
		"resolved":    resolved,
		"reason":      reason,
	})
}
