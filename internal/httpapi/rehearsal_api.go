package httpapi

import (
	"net/http"

	"task238-stagecue/internal/model"
)

// createRehearsal 处理 POST /api/rehearsals。
func (h *Handler) createRehearsal(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Show string `json:"show"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrInvalidClockSkew)
		return
	}
	rep, err := h.svc.Rehearsal.Create(body.Name, body.Show)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rep)
}

// listRehearsals 处理 GET /api/rehearsals。
func (h *Handler) listRehearsals(w http.ResponseWriter, r *http.Request) {
	reps, err := h.svc.Rehearsal.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reps)
}

// getRehearsal 处理 GET /api/rehearsals/:id。
func (h *Handler) getRehearsal(w http.ResponseWriter, r *http.Request, id string) {
	rep, err := h.svc.Rehearsal.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// addEvent 处理 POST /api/rehearsals/:id/events。
func (h *Handler) addEvent(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Source  string `json:"source"`
		Seq     int    `json:"seq"`
		Actor   string `json:"actor"`
		Role    string `json:"role"`
		Label   string `json:"label"`
		RawTs   int64  `json:"raw_ts"`
		Device  string `json:"device"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrInvalidClockSkew)
		return
	}
	ev, err := h.svc.Rehearsal.AddEvent(id, model.SourceKind(body.Source), body.Seq,
		model.ActorKind(body.Actor), model.EventRole(body.Role), body.Label, body.RawTs, body.Device)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}

// markPending 处理 POST /api/rehearsals/:id/mark-pending。
func (h *Handler) markPending(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Rehearsal.MarkPending(id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
}

// setSkew 处理 POST /api/rehearsals/:id/skew。
func (h *Handler) setSkew(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Source   string `json:"source"`
		SkewMs   int64  `json:"skew_ms"`
		Baseline bool   `json:"baseline"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrInvalidClockSkew)
		return
	}
	if err := h.svc.Align.SetSkew(id, model.SourceKind(body.Source), body.SkewMs, body.Baseline); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "skew_set"})
}

// review 处理 POST /api/rehearsals/:id/review。
func (h *Handler) review(w http.ResponseWriter, r *http.Request, id string) {
	res, err := h.svc.RunReview(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// timeline 处理 GET /api/rehearsals/:id/timeline。
func (h *Handler) timeline(w http.ResponseWriter, r *http.Request, id string) {
	events, err := h.svc.Align.Timeline(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// freeze 处理 POST /api/rehearsals/:id/freeze。
func (h *Handler) freeze(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Store.Rehearsals().SetState(id, model.StateFrozen, model.Now()); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "frozen"})
}
