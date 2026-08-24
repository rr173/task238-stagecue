package httpapi

import (
	"net/http"

	"task238-stagecue/internal/model"
)

// createCue 处理 POST /api/cues：作为 cue_log 源事件导入一条灯光提示。
func (h *Handler) createCue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RehearsalID string `json:"rehearsal_id"`
		Seq         int    `json:"seq"`
		Actor       string `json:"actor"`
		Role        string `json:"role"`
		Label       string `json:"label"`
		RawTs       int64  `json:"raw_ts"`
		Device      string `json:"device"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrInvalidClockSkew)
		return
	}
	ev, err := h.svc.Rehearsal.AddEvent(body.RehearsalID, model.SourceCueLog, body.Seq,
		model.ActorKind(body.Actor), model.EventRole(body.Role), body.Label, body.RawTs, body.Device)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}

// getCue 处理 GET /api/cues/:id。
func (h *Handler) getCue(w http.ResponseWriter, r *http.Request, id string) {
	ev, err := h.svc.Store.Events().Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

// setAnchor 处理 PATCH /api/cues/:id/anchor：调整触发锚点（重设 raw_ts 并重新校正）。
func (h *Handler) setAnchor(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		RawTs int64 `json:"raw_ts"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrInvalidClockSkew)
		return
	}
	ev, err := h.svc.Store.Events().Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if ev.RehearsalID == "" {
		writeError(w, model.ErrNotFound)
		return
	}
	ev.RawTimestamp = body.RawTs
	skew, err := h.svc.Store.Skews().SkewOf(ev.RehearsalID, ev.Source)
	if err != nil {
		writeError(w, err)
		return
	}
	ev.CorrectedAt = ev.RawTimestamp - skew
	if err := h.svc.Store.Events().UpsertBySeq(ev); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}
