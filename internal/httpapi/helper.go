package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task238-stagecue/internal/model"
)

var (
	errMethod  = errors.New("method not allowed")
	errNotFound = errors.New("route not found")
)

// writeJSON 写 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 将领域错误映射为 HTTP 状态并写错误体。
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch err {
	case model.ErrNotFound:
		status = http.StatusNotFound
	case model.ErrConflict:
		status = http.StatusConflict
	case model.ErrInvalidState:
		status = http.StatusConflict
	case model.ErrDuplicateCueSeq:
		status = http.StatusConflict
	case model.ErrNegativeDuration:
		status = http.StatusBadRequest
	case model.ErrUnknownRehearsal:
		status = http.StatusNotFound
	case model.ErrFrozenRehearsal:
		status = http.StatusConflict
	case model.ErrInvalidClockSkew:
		status = http.StatusBadRequest
	case model.ErrCircularRelation:
		status = http.StatusBadRequest
	case model.ErrMissingWaiver:
		status = http.StatusBadRequest
	case model.ErrReleasedPackage:
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// decodeJSON 解析请求体。
func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}
