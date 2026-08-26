package application

import (
	"context"
	"dialect-release/internal/audit"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"errors"
	"strings"
)

func (s *Service) CreateCase(ctx context.Context, input CreateCaseInput) (result Result, err error) {
	if strings.TrimSpace(input.RequestID) == "" {
		return Result{}, domain.Required("request_id")
	}
	if strings.TrimSpace(input.Actor) == "" {
		return Result{}, domain.Required("actor")
	}
	unlock := s.locks.lock("request:" + input.RequestID)
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
	if prior, lookup := tx.GetIdempotency(ctx, input.RequestID); lookup == nil {
		_ = tx.Rollback()
		return replay(prior, "create_case", "")
	} else if !errors.Is(lookup, store.ErrNotFound) {
		return Result{}, lookup
	}
	id, err := audit.NewID("case")
	if err != nil {
		return Result{}, err
	}
	now := s.now().UTC()
	c := &domain.ReleaseCase{ID: id, LanguageName: input.LanguageName, CollectionBatch: input.CollectionBatch, Owner: input.Owner, ReleaseLevel: input.ReleaseLevel, Status: domain.StatusDraft, Revision: 1, CreatedAt: now}
	if err = domain.ValidateNewCase(c); err != nil {
		return Result{}, err
	}
	if err = tx.InsertCase(ctx, c); err != nil {
		return Result{}, err
	}
	event := audit.NewEvent(c.ID, input.Actor, input.RequestID, "CASE_CREATED", 0, 1, now, map[string]any{"status": c.Status})
	if err = tx.AppendEvent(ctx, event); err != nil {
		return Result{}, err
	}
	result, err = response(201, CaseEnvelope{Case: c, Revision: 1})
	if err != nil {
		return Result{}, err
	}
	if err = tx.SaveIdempotency(ctx, store.IdempotencyRecord{RequestID: input.RequestID, CaseID: c.ID, Operation: "create_case", StatusCode: 201, Body: result.Body}); err != nil {
		return Result{}, err
	}
	if err = tx.Commit(); err != nil {
		return Result{}, err
	}
	return result, nil
}
