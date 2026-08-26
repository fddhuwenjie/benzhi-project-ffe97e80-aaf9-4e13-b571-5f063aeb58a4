package store

import (
	"context"
	"database/sql"
	"dialect-release/internal/domain"
	"errors"
	"fmt"
)

type sqliteTx struct{ tx *sql.Tx }

func (t *sqliteTx) Commit() error   { return t.tx.Commit() }
func (t *sqliteTx) Rollback() error { return t.tx.Rollback() }

func (t *sqliteTx) LoadCase(ctx context.Context, id string) (*domain.ReleaseCase, error) {
	return loadCase(ctx, t.tx, id)
}

func (t *sqliteTx) InsertCase(ctx context.Context, c *domain.ReleaseCase) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO release_cases(id,language_name,collection_batch,owner,release_level,status,revision,created_at,sealed_at) VALUES(?,?,?,?,?,?,?,?,?)`, c.ID, c.LanguageName, c.CollectionBatch, c.Owner, c.ReleaseLevel, c.Status, c.Revision, c.CreatedAt.UTC().Format(timeFormat), nullableTime(c.SealedAt))
	if err != nil {
		return fmt.Errorf("新增个案: %w", err)
	}
	return t.saveChildren(ctx, c)
}

func (t *sqliteTx) SaveCase(ctx context.Context, c *domain.ReleaseCase, expected int64) error {
	result, err := t.tx.ExecContext(ctx, `UPDATE release_cases SET language_name=?,collection_batch=?,owner=?,release_level=?,status=?,revision=?,sealed_at=? WHERE id=? AND revision=?`, c.LanguageName, c.CollectionBatch, c.Owner, c.ReleaseLevel, c.Status, c.Revision, nullableTime(c.SealedAt), c.ID, expected)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ErrConflict
	}
	for _, query := range []string{`DELETE FROM sensitivity_findings WHERE asset_id IN (SELECT id FROM recording_assets WHERE case_id=?)`, `DELETE FROM recording_assets WHERE case_id=?`, `DELETE FROM consent_grants WHERE case_id=?`, `DELETE FROM steward_reviews WHERE case_id=?`} {
		if _, err := t.tx.ExecContext(ctx, query, c.ID); err != nil {
			return err
		}
	}
	return t.saveChildren(ctx, c)
}

func (t *sqliteTx) saveChildren(ctx context.Context, c *domain.ReleaseCase) error {
	for _, grant := range c.Consents {
		uses, _ := encode(grant.PermittedUses)
		regions, _ := encode(grant.RegionLimits)
		if _, err := t.tx.ExecContext(ctx, `INSERT INTO consent_grants VALUES(?,?,?,?,?,?,?,?)`, grant.ID, c.ID, grant.ParticipantCode, grant.EvidenceRef, uses, regions, nullableTime(grant.ValidUntil), nullableTime(grant.WithdrawnAt)); err != nil {
			return err
		}
	}
	for _, asset := range c.Assets {
		participants, _ := encode(asset.ParticipantCodes)
		if _, err := t.tx.ExecContext(ctx, `INSERT INTO recording_assets VALUES(?,?,?,?,?,?,?,?)`, asset.ID, c.ID, asset.StableKey, asset.Summary, asset.DurationMS, asset.CapturedOn, participants, asset.ContentSHA256); err != nil {
			return err
		}
		for _, finding := range asset.Findings {
			if _, err := t.tx.ExecContext(ctx, `INSERT INTO sensitivity_findings VALUES(?,?,?,?,?,?,?,?,?)`, finding.ID, asset.ID, finding.StartMS, finding.EndMS, finding.Category, finding.Severity, finding.Disposition, finding.TreatmentNote, finding.Status); err != nil {
				return err
			}
		}
	}
	for _, review := range c.Reviews {
		reasons, _ := encode(review.ReasonCodes)
		if _, err := t.tx.ExecContext(ctx, `INSERT INTO steward_reviews(id,case_id,round,reviewer,decision,reason_codes,comment,decided_at,finding_count) VALUES(?,?,?,?,?,?,?,?,?)`, review.ID, c.ID, review.Round, review.Reviewer, review.Decision, reasons, review.Comment, review.DecidedAt.UTC().Format(timeFormat), review.FindingCount); err != nil {
			return err
		}
	}
	return nil
}

func (t *sqliteTx) AppendEvent(ctx context.Context, event domain.AuditEvent) error {
	details, _ := encode(event.Details)
	_, err := t.tx.ExecContext(ctx, `INSERT INTO audit_events(case_id,actor,request_id,before_revision,after_revision,event_type,occurred_at,details) VALUES(?,?,?,?,?,?,?,?)`, event.CaseID, event.Actor, event.RequestID, event.BeforeRevision, event.AfterRevision, event.EventType, event.OccurredAt.UTC().Format(timeFormat), details)
	return err
}

func (t *sqliteTx) SaveManifest(ctx context.Context, manifest *domain.ReleaseManifest) error {
	assets, _ := encode(manifest.AssetEntries)
	constraints, _ := encode(manifest.ConstraintSummary)
	_, err := t.tx.ExecContext(ctx, `INSERT INTO release_manifests VALUES(?,?,?,?,?,?,?,?)`, manifest.ID, manifest.CaseID, manifest.CaseRevision, assets, constraints, manifest.CanonicalJSON, manifest.SHA256, manifest.SealedAt.UTC().Format(timeFormat))
	return err
}

func (t *sqliteTx) GetIdempotency(ctx context.Context, requestID string) (*IdempotencyRecord, error) {
	var record IdempotencyRecord
	err := t.tx.QueryRowContext(ctx, `SELECT request_id,case_id,operation,status_code,body FROM idempotency_records WHERE request_id=?`, requestID).Scan(&record.RequestID, &record.CaseID, &record.Operation, &record.StatusCode, &record.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &record, err
}

func (t *sqliteTx) SaveIdempotency(ctx context.Context, record IdempotencyRecord) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO idempotency_records(request_id,case_id,operation,status_code,body) VALUES(?,?,?,?,?)`, record.RequestID, record.CaseID, record.Operation, record.StatusCode, record.Body)
	return err
}
