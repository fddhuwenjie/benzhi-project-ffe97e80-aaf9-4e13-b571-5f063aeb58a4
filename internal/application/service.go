package application

import (
	"context"
	"crypto/sha256"
	"dialect-release/internal/audit"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repo  store.Repository
	locks *keyedLocks
	now   func() time.Time
}

func New(repo store.Repository) *Service {
	return &Service{repo: repo, locks: newKeyedLocks(), now: time.Now}
}

func validateMeta(meta CommandMeta) error {
	if strings.TrimSpace(meta.RequestID) == "" {
		return domain.Required("request_id")
	}
	if strings.TrimSpace(meta.Actor) == "" {
		return domain.Required("actor")
	}
	if meta.ExpectedRevision < 1 {
		return domain.NewRuleError("validation_error", "expected_revision 必须大于 0")
	}
	return nil
}

func response(status int, payload any) (Result, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, err
	}
	return Result{StatusCode: status, Body: body}, nil
}

func replay(record *store.IdempotencyRecord, operation, caseID, payloadHash string) (Result, error) {
	if record.Operation != operation || (caseID != "" && record.CaseID != caseID) || payloadHash != record.PayloadHash {
		return Result{}, domain.NewRuleError("request_id_reused", "request_id 已用于其他业务请求")
	}
	return Result{StatusCode: record.StatusCode, Body: record.Body, Replayed: true}, nil
}

// payloadHash 计算请求业务载荷的稳定摘要，用于幂等比对。
// 它由操作名、路径参数与业务输入的规范化 JSON 组成，
// 当同一 request_id 再次提交时，任何业务字段差异都应产生不同摘要。
func payloadHash(operation string, pathValues []string, input any) string {
	h := sha256.New()
	h.Write([]byte(operation))
	h.Write([]byte{0})
	for _, v := range pathValues {
		h.Write([]byte(v))
		h.Write([]byte{0})
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	h.Write(encoded)
	return hex.EncodeToString(h.Sum(nil))
}

func ruleFailureResult(err error) (Result, bool) {
	var rule *domain.RuleError
	if !errors.As(err, &rule) {
		return Result{}, false
	}
	status := 422
	switch rule.Code {
	case "revision_conflict", "case_sealed", "request_id_reused", "invalid_status", "finding_already_closed":
		status = 409
	}
	result, marshalErr := response(status, map[string]any{"error": map[string]string{"code": rule.Code, "message": rule.Message}})
	if marshalErr != nil {
		return Result{}, false
	}
	return result, true
}

func (s *Service) mutate(ctx context.Context, caseID, operation, eventType string, pathValues []string, meta CommandMeta, input any, apply func(*domain.ReleaseCase, time.Time) (*domain.ReleaseManifest, error)) (result Result, err error) {
	if err := validateMeta(meta); err != nil {
		return Result{}, err
	}
	hash := payloadHash(operation, pathValues, input)
	unlock := s.locks.lock(caseID)
	defer unlock()
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if prior, lookup := tx.GetIdempotency(ctx, meta.RequestID); lookup == nil {
		_ = tx.Rollback()
		return replay(prior, operation, caseID, hash)
	} else if !errors.Is(lookup, store.ErrNotFound) {
		return Result{}, lookup
	}
	c, err := tx.LoadCase(ctx, caseID)
	if err != nil {
		return Result{}, err
	}
	if c.Revision != meta.ExpectedRevision {
		return Result{}, domain.NewRuleError("revision_conflict", fmt.Sprintf("expected_revision=%d，当前 revision=%d", meta.ExpectedRevision, c.Revision))
	}
	before := c.Revision
	now := s.now().UTC()
	manifest, err := apply(c, now)
	if err != nil {
		if failed, ok := ruleFailureResult(err); ok {
			if saveErr := tx.SaveIdempotency(ctx, store.IdempotencyRecord{RequestID: meta.RequestID, CaseID: c.ID, Operation: operation, PayloadHash: hash, StatusCode: failed.StatusCode, Body: failed.Body}); saveErr == nil {
				if commitErr := tx.Commit(); commitErr != nil {
					return Result{}, commitErr
				}
			}
		}
		return Result{}, err
	}
	c.RefreshAuthorizationSummary(now)
	c.Revision++
	if err = tx.SaveCase(ctx, c, before); err != nil {
		return Result{}, err
	}
	if manifest != nil {
		if err = tx.SaveManifest(ctx, manifest); err != nil {
			return Result{}, err
		}
	}
	event := audit.NewEvent(c.ID, meta.Actor, meta.RequestID, eventType, before, c.Revision, now, map[string]any{"status": c.Status})
	if err = tx.AppendEvent(ctx, event); err != nil {
		return Result{}, err
	}
	payload := any(CaseEnvelope{Case: c, Revision: c.Revision})
	status := 200
	if manifest != nil {
		payload = ManifestEnvelope{Manifest: manifest, Revision: c.Revision}
	}
	result, err = response(status, payload)
	if err != nil {
		return Result{}, err
	}
	if err = tx.SaveIdempotency(ctx, store.IdempotencyRecord{RequestID: meta.RequestID, CaseID: c.ID, Operation: operation, PayloadHash: hash, StatusCode: status, Body: result.Body}); err != nil {
		return Result{}, err
	}
	if err = tx.Commit(); err != nil {
		return Result{}, err
	}
	return result, nil
}
