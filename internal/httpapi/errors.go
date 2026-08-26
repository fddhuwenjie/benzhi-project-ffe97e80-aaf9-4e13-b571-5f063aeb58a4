package httpapi

import (
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"errors"
	"log"
	"net/http"
)

type requestError struct{ code, message string }

func (e requestError) Error() string { return e.message }

func handleError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "服务内部错误"
	var req requestError
	var rule *domain.RuleError
	switch {
	case errors.As(err, &req):
		code = req.code
		message = req.message
		if code == "unsupported_media_type" {
			status = http.StatusUnsupportedMediaType
		} else if code == "request_too_large" {
			status = http.StatusRequestEntityTooLarge
		} else {
			status = http.StatusBadRequest
		}
	case errors.As(err, &rule):
		code = rule.Code
		message = rule.Message
		status = http.StatusUnprocessableEntity
		if code == "revision_conflict" || code == "case_sealed" || code == "request_id_reused" || code == "invalid_status" {
			status = http.StatusConflict
		}
	case errors.Is(err, store.ErrNotFound):
		code = "not_found"
		message = "请求的资源不存在"
		status = http.StatusNotFound
	case errors.Is(err, store.ErrConflict):
		code = "revision_conflict"
		message = "修订已被其他请求更新"
		status = http.StatusConflict
	default:
		log.Printf("HTTP 请求失败: %v", err)
	}
	response := errorResponse{}
	response.Error.Code = code
	response.Error.Message = message
	writeJSON(w, status, response)
}
