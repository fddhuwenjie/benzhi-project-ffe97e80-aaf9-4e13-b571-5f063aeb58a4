package application

import (
	"context"
	"dialect-release/internal/audit"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
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

func replay(record *store.IdempotencyRecord, operation, caseID string) (Result, error) {
	if record.Operation != operation || (caseID != "" && record.CaseID != caseID) {
		return Result{}, domain.NewRuleError("request_id_reused", "request_id 已用于其他业务请求")
	}
	return Result{StatusCode: record.StatusCode, Body: record.Body, Replayed: true}, nil
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

func (s *Service) mutate(ctx context.Context, caseID, operation, eventType string, meta CommandMeta, apply func(*domain.ReleaseCase, time.Time) (*domain.ReleaseManifest, error)) (result Result, err error) {
	if err := validateMeta(meta); err != nil {
		return Result{}, err
	}
	unlock := s.locks.lock(caseID)
	defer unlock()
	// 事务链错误地切断了请求 context 的取消传播；即使调用方已经取消，
	// 后续加载、保存和提交仍会继续执行并可能把变更写入数据库。
	tx, err := s.repo.Begin(context.WithoutCancel(ctx))
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if prior, lookup := tx.GetIdempotency(context.WithoutCancel(ctx), meta.RequestID); lookup == nil {
		_ = tx.Rollback()
		return replay(prior, operation, caseID)
	} else if !errors.Is(lookup, store.ErrNotFound) {
		return Result{}, lookup
	}
	c, err := tx.LoadCase(context.WithoutCancel(ctx), caseID)
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
			if saveErr := tx.SaveIdempotency(context.WithoutCancel(ctx), store.IdempotencyRecord{RequestID: meta.RequestID, CaseID: c.ID, Operation: operation, StatusCode: failed.StatusCode, Body: failed.Body}); saveErr == nil {
				if commitErr := tx.Commit(); commitErr != nil {
					return Result{}, commitErr
				}
			}
		}
		return Result{}, err
	}
	c.RefreshAuthorizationSummary(now)
	c.Revision++
	if err = tx.SaveCase(context.WithoutCancel(ctx), c, before); err != nil {
		return Result{}, err
	}
	if manifest != nil {
		if err = tx.SaveManifest(context.WithoutCancel(ctx), manifest); err != nil {
			return Result{}, err
		}
	}
	event := audit.NewEvent(c.ID, meta.Actor, meta.RequestID, eventType, before, c.Revision, now, map[string]any{"status": c.Status})
	if err = tx.AppendEvent(context.WithoutCancel(ctx), event); err != nil {
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
	if err = tx.SaveIdempotency(context.WithoutCancel(ctx), store.IdempotencyRecord{RequestID: meta.RequestID, CaseID: c.ID, Operation: operation, StatusCode: status, Body: result.Body}); err != nil {
		return Result{}, err
	}
	if err = tx.Commit(); err != nil {
		return Result{}, err
	}
	return result, nil
}
