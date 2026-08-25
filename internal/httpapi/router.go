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
	// Use method-aware patterns so every public capability is explicit in the
	// mux and unsupported methods are rejected before reaching a subrouter.
	// Go 1.22+ ServeMux supplies the named path values used below.
	mux.HandleFunc("GET /api/rehearsals", h.listRehearsals)
	mux.HandleFunc("POST /api/rehearsals", h.createRehearsal)
	mux.HandleFunc("GET /api/rehearsals/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.getRehearsal(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/rehearsals/{id}/events", func(w http.ResponseWriter, r *http.Request) {
		h.addEvent(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/rehearsals/{id}/skew", func(w http.ResponseWriter, r *http.Request) {
		h.setSkew(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/rehearsals/{id}/review", func(w http.ResponseWriter, r *http.Request) {
		h.review(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/rehearsals/{id}/timeline", func(w http.ResponseWriter, r *http.Request) {
		h.timeline(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/rehearsals/{id}/mark-pending", func(w http.ResponseWriter, r *http.Request) {
		h.markPending(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/rehearsals/{id}/freeze", func(w http.ResponseWriter, r *http.Request) {
		h.freeze(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/rehearsals/{id}/conflicts", func(w http.ResponseWriter, r *http.Request) {
		h.listConflicts(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/cues", h.createCue)
	mux.HandleFunc("GET /api/cues/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.getCue(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("PATCH /api/cues/{id}/anchor", func(w http.ResponseWriter, r *http.Request) {
		h.setAnchor(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/constraints", h.listConstraints)
	mux.HandleFunc("POST /api/constraints", h.createConstraint)
	mux.HandleFunc("POST /api/constraints/{id}/activate", func(w http.ResponseWriter, r *http.Request) {
		h.activateConstraint(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/constraints/{id}/revoke", func(w http.ResponseWriter, r *http.Request) {
		h.revokeConstraint(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/conflicts/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.getConflict(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/conflicts/{id}/waiver", func(w http.ResponseWriter, r *http.Request) {
		h.waiverConflict(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/conflicts/{id}/resolution", func(w http.ResponseWriter, r *http.Request) {
		h.waiverResolution(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/waivers", h.listWaivers)
	mux.HandleFunc("GET /api/packages", h.listPackages)
	mux.HandleFunc("POST /api/packages", h.draftPackage)
	mux.HandleFunc("GET /api/packages/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.getPackage(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/packages/{id}/publish", func(w http.ResponseWriter, r *http.Request) {
		h.publishPackage(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /api/packages/{id}/supersede", func(w http.ResponseWriter, r *http.Request) {
		h.supersedePackage(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /api/selfcheck", h.selfCheck)

	// Keep the legacy subrouters as a narrow fallback for paths not covered by
	// the explicit method-aware registrations (for example malformed paths).
	mux.HandleFunc("/api/rehearsals/", h.rehearsalSub)
	mux.HandleFunc("/api/cues/", h.cueSub)
	mux.HandleFunc("/api/constraints/", h.constraintSub)
	mux.HandleFunc("/api/conflicts/", h.conflictSub)
	mux.HandleFunc("/api/packages/", h.packageSub)
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
	case len(parts) >= 2 && parts[1] == "waiver" && r.Method == http.MethodPost:
		h.waiverConflict(w, r, id)
	case len(parts) >= 2 && parts[1] == "resolution" && r.Method == http.MethodGet:
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
	case len(parts) >= 2 && parts[1] == "publish" && r.Method == http.MethodPost:
		h.publishPackage(w, r, id)
	case len(parts) >= 2 && parts[1] == "supersede" && r.Method == http.MethodPost:
		h.supersedePackage(w, r, id)
	default:
		writeError(w, errNotFound)
	}
}
