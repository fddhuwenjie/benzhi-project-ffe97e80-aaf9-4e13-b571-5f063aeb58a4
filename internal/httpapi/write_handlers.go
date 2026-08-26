package httpapi

import (
	"dialect-release/internal/application"
	"net/http"
)

func (a *API) CreateCase(w http.ResponseWriter, r *http.Request) {
	var input application.CreateCaseInput
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.CreateCase(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) AddConsent(w http.ResponseWriter, r *http.Request) {
	var input application.ConsentInput
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.AddConsent(r.Context(), r.PathValue("caseID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) WithdrawConsent(w http.ResponseWriter, r *http.Request) {
	var input application.WithdrawConsentInput
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.WithdrawConsent(r.Context(), r.PathValue("caseID"), r.PathValue("consentID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) AddAsset(w http.ResponseWriter, r *http.Request) {
	var input application.AssetInput
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.AddAsset(r.Context(), r.PathValue("caseID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) AddAssetBatch(w http.ResponseWriter, r *http.Request) {
	var input application.AssetBatchInput
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	if len(input.Assets) > 100 {
		handleError(w, requestError{"validation_error", "assets 每批最多 100 项"})
		return
	}
	result, err := a.service.AddAssetBatch(r.Context(), r.PathValue("caseID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) AddFinding(w http.ResponseWriter, r *http.Request) {
	var input application.FindingInput
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.AddFinding(r.Context(), r.PathValue("caseID"), r.PathValue("assetID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) CloseFinding(w http.ResponseWriter, r *http.Request) {
	var input application.CloseFindingInput
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.CloseFinding(r.Context(), r.PathValue("caseID"), r.PathValue("findingID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) SubmitReview(w http.ResponseWriter, r *http.Request) {
	var input application.CommandMeta
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.Submit(r.Context(), r.PathValue("caseID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) StewardReview(w http.ResponseWriter, r *http.Request) {
	var input application.ReviewInput
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.Review(r.Context(), r.PathValue("caseID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}

func (a *API) ApproveCase(w http.ResponseWriter, r *http.Request) {
	var input application.CommandMeta
	if err := decodeJSON(w, r, &input); err != nil {
		handleError(w, err)
		return
	}
	result, err := a.service.ApproveAndSeal(r.Context(), r.PathValue("caseID"), input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeRaw(w, result.StatusCode, result.Body, result.Replayed)
}
