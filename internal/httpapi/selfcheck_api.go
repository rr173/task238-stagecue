package httpapi

import (
	"net/http"

	"task238-stagecue/internal/model"
)

// selfCheck 处理 GET /api/selfcheck：返回服务健康与模块可用性。
func (h *Handler) selfCheck(w http.ResponseWriter, r *http.Request) {
	// 验证 store 可访问
	if _, err := h.svc.Store.Rehearsals().List(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":           true,
		"service":      "stagecue",
		"requirement":  "REQ-20260823-101",
		"model_states": []model.State{
			model.StateImporting, model.StatePending, model.StateReviewable, model.StateFrozen,
			model.StateValid, model.StateEarly, model.StateLate, model.StateConflict,
			model.StateDraft, model.StateActive, model.StateWaived, model.StateRevoked,
			model.StatePublished, model.StateSuperseded,
		},
	})
}
