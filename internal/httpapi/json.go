package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxRequestBytes = 1 << 20

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return requestError{"unsupported_media_type", "Content-Type 必须是 application/json"}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return requestError{"request_too_large", "请求体超过 1 MiB 限制"}
		}
		return requestError{"invalid_json", "JSON 请求体无效或包含未知字段"}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return requestError{"invalid_json", "请求体只能包含一个 JSON 对象"}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeRaw(w http.ResponseWriter, status int, body []byte, replayed bool) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if replayed {
		w.Header().Set("Idempotent-Replayed", "true")
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
