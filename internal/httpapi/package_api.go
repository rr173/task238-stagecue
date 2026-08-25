package httpapi

import (
	"net/http"

	"task238-stagecue/internal/model"
)

// draftPackage 处理 POST /api/packages：为某演练创建草稿提示包（计算不可变快照）。
func (h *Handler) draftPackage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RehearsalID string `json:"rehearsal_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrUnknownRehearsal)
		return
	}
	pkg, err := h.svc.CuePkg.Draft(body.RehearsalID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pkg)
}

// getPackage 处理 GET /api/packages/:id。
func (h *Handler) getPackage(w http.ResponseWriter, r *http.Request, id string) {
	pkg, err := h.svc.CuePkg.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

// publishPackage 处理 POST /api/packages/:id/publish：要求无未解决冲突。
func (h *Handler) publishPackage(w http.ResponseWriter, r *http.Request, id string) {
	if r.URL.Query().Get("force") == "true" {
		id = r.URL.Query().Get("package_id")
	}
	pkg, err := h.svc.CuePkg.Publish(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

// supersedePackage 处理 POST /api/packages/:id/supersede：用新版本替代旧版本。
func (h *Handler) supersedePackage(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		NewPackageID string `json:"new_package_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, model.ErrReleasedPackage)
		return
	}
	if err := h.svc.CuePkg.Supersede(id, body.NewPackageID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "superseded"})
}

// listPackages 处理 GET /api/packages?rehearsal_id=。
func (h *Handler) listPackages(w http.ResponseWriter, r *http.Request) {
	rid := r.URL.Query().Get("rehearsal_id")
	if rid == "" {
		writeError(w, model.ErrUnknownRehearsal)
		return
	}
	pkgs, err := h.svc.CuePkg.List(rid)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pkgs)
}
