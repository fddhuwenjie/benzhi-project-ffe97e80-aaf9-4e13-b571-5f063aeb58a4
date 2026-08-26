package httpapi

import (
	"dialect-release/internal/application"
	"net/http"
)

type API struct {
	service *application.Service
	mux     *http.ServeMux
}

func New(service *application.Service) *API {
	a := &API{service: service, mux: http.NewServeMux()}
	a.routes()
	return a
}

func (a *API) Handler() http.Handler {
	return recoverMiddleware(contentMiddleware(a.mux))
}

func (a *API) routes() {
	a.mux.HandleFunc("GET /healthz", a.Health)
	a.mux.HandleFunc("GET /readyz", a.Ready)
	a.mux.HandleFunc("POST /api/v1/cases", a.CreateCase)
	a.mux.HandleFunc("GET /api/v1/cases", a.ListCases)
	a.mux.HandleFunc("GET /api/v1/cases/{caseID}", a.GetCase)
	a.mux.HandleFunc("POST /api/v1/cases/{caseID}/consents", a.AddConsent)
	a.mux.HandleFunc("PATCH /api/v1/cases/{caseID}/consents/{consentID}/withdraw", a.WithdrawConsent)
	a.mux.HandleFunc("POST /api/v1/cases/{caseID}/assets", a.AddAsset)
	a.mux.HandleFunc("POST /api/v1/cases/{caseID}/assets/batch", a.AddAssetBatch)
	a.mux.HandleFunc("POST /api/v1/cases/{caseID}/assets/{assetID}/findings", a.AddFinding)
	a.mux.HandleFunc("PATCH /api/v1/cases/{caseID}/findings/{findingID}", a.CloseFinding)
	a.mux.HandleFunc("POST /api/v1/cases/{caseID}/submit", a.SubmitReview)
	a.mux.HandleFunc("POST /api/v1/cases/{caseID}/reviews", a.StewardReview)
	a.mux.HandleFunc("POST /api/v1/cases/{caseID}/approve", a.ApproveCase)
	a.mux.HandleFunc("GET /api/v1/cases/{caseID}/timeline", a.GetTimeline)
	a.mux.HandleFunc("GET /api/v1/cases/{caseID}/manifest", a.GetManifest)
}
