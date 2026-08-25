package httpapi

import (
	"net/http"

	"task238-stagecue/internal/model"
)

// createConstraint 处理 POST /api/constraints。
func (h *Handler) createConstraint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RehearsalID string `json:"rehearsal_id"`
		Name        string `json:"name"`
		Actor       string `json:"actor"`
		MinGapMs    int64  `json:"min_gap_ms"`
		Note        string `json:"note"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrNegativeDuration)
		return
	}
	ct, err := h.svc.Constraint.Add(body.RehearsalID, body.Name, model.ActorKind(body.Actor),
		body.MinGapMs, body.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ct)
}

// activateConstraint 处理 POST /api/constraints/:id/activate。
func (h *Handler) activateConstraint(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Constraint.Activate(id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

// revokeConstraint 处理 POST /api/constraints/:id/revoke。
func (h *Handler) revokeConstraint(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Constraint.Revoke(id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// listConstraints 处理 GET /api/constraints?rehearsal_id=。
func (h *Handler) listConstraints(w http.ResponseWriter, r *http.Request) {
	rid := r.URL.Query().Get("rehearsal_id")
	if rid == "" {
		writeError(w, model.ErrUnknownRehearsal)
		return
	}
	cts, err := h.svc.Constraint.List(rid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cts)
}
