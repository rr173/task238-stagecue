package httpapi

import (
	"net/http"
	"strings"

	"task238-stagecue/internal/service"
)

// Handler 持有编排服务，注册全部 /api 路由。
type Handler struct {
	svc *service.Service
}

// NewHandler 构造 HTTP Handler。
func NewHandler(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Routes 返回已注册路由的 *http.ServeMux。
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/rehearsals", h.wrapList(h.listRehearsals, h.createRehearsal))
	mux.HandleFunc("/api/rehearsals/", h.rehearsalSub)
	mux.HandleFunc("/api/cues", h.wrapOne(h.createCue))
	mux.HandleFunc("/api/cues/", h.cueSub)
	mux.HandleFunc("/api/constraints", h.wrapList(h.listConstraints, h.createConstraint))
	mux.HandleFunc("/api/constraints/", h.constraintSub)
	mux.HandleFunc("/api/conflicts/", h.conflictSub)
	mux.HandleFunc("/api/waivers", h.wrapList(h.listWaivers, nil))
	mux.HandleFunc("/api/packages", h.wrapList(h.listPackages, h.draftPackage))
	mux.HandleFunc("/api/packages/", h.packageSub)
	mux.HandleFunc("/api/selfcheck", h.wrapOne(h.selfCheck))
	return mux
}

// wrapList 处理集合路由：GET→listFn，POST→createFn（createFn 可为 nil）。
func (h *Handler) wrapList(listFn, createFn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listFn(w, r)
		case http.MethodPost:
			if createFn == nil {
				writeError(w, errMethod)
				return
			}
			createFn(w, r)
		default:
			writeError(w, errMethod)
		}
	}
}

// wrapOne 单方法路由。
func (h *Handler) wrapOne(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, errMethod)
			return
		}
		fn(w, r)
	}
}

func (h *Handler) rehearsalSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/rehearsals/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, errNotFound)
		return
	}
	switch {
	case len(parts) == 1:
		if r.Method == http.MethodGet {
			h.getRehearsal(w, r, id)
		} else {
			writeError(w, errMethod)
		}
	case parts[1] == "events" && r.Method == http.MethodPost:
		h.addEvent(w, r, id)
	case parts[1] == "skew" && r.Method == http.MethodPost:
		h.setSkew(w, r, id)
	case parts[1] == "review" && r.Method == http.MethodPost:
		h.review(w, r, id)
	case parts[1] == "timeline" && r.Method == http.MethodGet:
		h.timeline(w, r, id)
	case parts[1] == "mark-pending" && r.Method == http.MethodPost:
		h.markPending(w, r, id)
	case parts[1] == "freeze" && r.Method == http.MethodPost:
		h.freeze(w, r, id)
	case parts[1] == "conflicts" && r.Method == http.MethodGet:
		h.listConflicts(w, r, id)
	default:
		writeError(w, errNotFound)
	}
}

func (h *Handler) cueSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/cues/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, errNotFound)
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		h.getCue(w, r, id)
	case len(parts) >= 2 && parts[1] == "anchor" && r.Method == http.MethodPatch:
		h.setAnchor(w, r, id)
	default:
		writeError(w, errNotFound)
	}
}

func (h *Handler) constraintSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/constraints/")
	id := strings.Split(rest, "/")[0]
	if id == "" {
		writeError(w, errNotFound)
		return
	}
	switch {
	case strings.HasSuffix(rest, "/activate") && r.Method == http.MethodPost:
		h.activateConstraint(w, r, id)
	case strings.HasSuffix(rest, "/revoke") && r.Method == http.MethodPost:
		h.revokeConstraint(w, r, id)
	default:
		writeError(w, errNotFound)
	}
}

func (h *Handler) conflictSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/conflicts/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, errNotFound)
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		h.getConflict(w, r, id)
	case parts[1] == "waiver" && r.Method == http.MethodPost:
		h.waiverConflict(w, r, id)
	case parts[1] == "resolution" && r.Method == http.MethodGet:
		h.waiverResolution(w, r, id)
	default:
		writeError(w, errNotFound)
	}
}

func (h *Handler) packageSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/packages/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, errNotFound)
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		h.getPackage(w, r, id)
	case parts[1] == "publish" && r.Method == http.MethodPost:
		h.publishPackage(w, r, id)
	case parts[1] == "supersede" && r.Method == http.MethodPost:
		h.supersedePackage(w, r, id)
	default:
		writeError(w, errNotFound)
	}
}
