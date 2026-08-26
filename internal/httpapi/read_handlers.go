package httpapi

import (
	"dialect-release/internal/application"
	"net/http"
	"strconv"
)

func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (a *API) Ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) GetCase(w http.ResponseWriter, r *http.Request) {
	value, err := a.service.GetCase(r.Context(), r.PathValue("caseID"))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"case": value, "revision": value.Revision})
}

func (a *API) ListCases(w http.ResponseWriter, r *http.Request) {
	query := application.CaseListQuery{Status: r.URL.Query().Get("status"), ReleaseLevel: r.URL.Query().Get("release_level"), Owner: r.URL.Query().Get("owner"), CreatedFrom: r.URL.Query().Get("created_from"), CreatedTo: r.URL.Query().Get("created_to")}
	values, err := a.service.ListCasesQuery(r.Context(), query)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, values)
}

func (a *API) GetTimeline(w http.ResponseWriter, r *http.Request) {
	limit, err := parseIntQuery(r, "limit", 0)
	if err != nil {
		handleError(w, err)
		return
	}
	after, err := parseInt64Query(r, "after_revision")
	if err != nil {
		handleError(w, err)
		return
	}
	before, err := parseInt64Query(r, "before_revision")
	if err != nil {
		handleError(w, err)
		return
	}
	if limit < 0 || limit > 200 {
		handleError(w, requestError{"validation_error", "limit 必须位于 1..200"})
		return
	}
	events, err := a.service.TimelinePage(r.Context(), r.PathValue("caseID"), application.TimelineQuery{Limit: limit, AfterRevision: after, BeforeRevision: before})
	if err != nil {
		handleError(w, err)
		return
	}
	next := ""
	if limit > 0 && len(events) == limit {
		next = strconv.FormatInt(events[len(events)-1].AfterRevision, 10)
	}
	writeJSON(w, http.StatusOK, application.TimelineResult{Events: events, Count: len(events), NextCursor: next})
}

func parseIntQuery(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, requestError{"validation_error", name + " 必须是整数"}
	}
	return value, nil
}
func parseInt64Query(r *http.Request, name string) (int64, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, requestError{"validation_error", name + " 必须是非负整数"}
	}
	return value, nil
}

func (a *API) GetManifest(w http.ResponseWriter, r *http.Request) {
	manifest, err := a.service.Manifest(r.Context(), r.PathValue("caseID"))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"manifest": manifest})
}
