package store

import (
	"context"
	"database/sql"
	"dialect-release/internal/domain"
	"errors"
	"fmt"
	"strings"
	"time"
)

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r *SQLiteRepository) GetCase(ctx context.Context, id string) (*domain.ReleaseCase, error) {
	return loadCase(ctx, r.db, id)
}

func loadCase(ctx context.Context, q queryer, id string) (*domain.ReleaseCase, error) {
	var c domain.ReleaseCase
	var status, created string
	var sealed sql.NullString
	err := q.QueryRowContext(ctx, `SELECT id,language_name,collection_batch,owner,release_level,status,revision,created_at,sealed_at FROM release_cases WHERE id=?`, id).Scan(&c.ID, &c.LanguageName, &c.CollectionBatch, &c.Owner, &c.ReleaseLevel, &status, &c.Revision, &created, &sealed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Status = domain.CaseStatus(status)
	c.CreatedAt, err = parseTime(created)
	if err != nil {
		return nil, err
	}
	c.SealedAt, err = parseNullableTime(sealed)
	if err != nil {
		return nil, err
	}
	if err := loadConsents(ctx, q, &c); err != nil {
		return nil, err
	}
	if err := loadAssets(ctx, q, &c); err != nil {
		return nil, err
	}
	if err := loadReviews(ctx, q, &c); err != nil {
		return nil, err
	}
	c.RefreshAuthorizationSummary(time.Now().UTC())
	return &c, nil
}

func loadConsents(ctx context.Context, q queryer, c *domain.ReleaseCase) error {
	rows, err := q.QueryContext(ctx, `SELECT id,participant_code,evidence_ref,permitted_uses,region_limits,valid_until,withdrawn_at FROM consent_grants WHERE case_id=? ORDER BY participant_code`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var grant domain.ConsentGrant
		var uses, regions string
		var valid, withdrawn sql.NullString
		grant.CaseID = c.ID
		if err := rows.Scan(&grant.ID, &grant.ParticipantCode, &grant.EvidenceRef, &uses, &regions, &valid, &withdrawn); err != nil {
			return err
		}
		if err := decode(uses, &grant.PermittedUses); err != nil {
			return err
		}
		if err := decode(regions, &grant.RegionLimits); err != nil {
			return err
		}
		grant.ValidUntil, err = parseNullableTime(valid)
		if err != nil {
			return err
		}
		grant.WithdrawnAt, err = parseNullableTime(withdrawn)
		if err != nil {
			return err
		}
		c.Consents = append(c.Consents, grant)
	}
	return rows.Err()
}

func loadAssets(ctx context.Context, q queryer, c *domain.ReleaseCase) error {
	rows, err := q.QueryContext(ctx, `SELECT id,stable_key,summary,duration_ms,captured_on,participant_codes,content_sha256 FROM recording_assets WHERE case_id=? ORDER BY stable_key`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var asset domain.RecordingAsset
		var participants string
		asset.CaseID = c.ID
		if err := rows.Scan(&asset.ID, &asset.StableKey, &asset.Summary, &asset.DurationMS, &asset.CapturedOn, &participants, &asset.ContentSHA256); err != nil {
			return err
		}
		if err := decode(participants, &asset.ParticipantCodes); err != nil {
			return err
		}
		findingRows, err := q.QueryContext(ctx, `SELECT id,start_ms,end_ms,category,severity,disposition,treatment_note,status FROM sensitivity_findings WHERE asset_id=? ORDER BY start_ms,id`, asset.ID)
		if err != nil {
			return err
		}
		for findingRows.Next() {
			var f domain.SensitivityFinding
			f.AssetID = asset.ID
			if err := findingRows.Scan(&f.ID, &f.StartMS, &f.EndMS, &f.Category, &f.Severity, &f.Disposition, &f.TreatmentNote, &f.Status); err != nil {
				findingRows.Close()
				return err
			}
			asset.Findings = append(asset.Findings, f)
		}
		if err := findingRows.Close(); err != nil {
			return err
		}
		c.Assets = append(c.Assets, asset)
	}
	return rows.Err()
}

func loadReviews(ctx context.Context, q queryer, c *domain.ReleaseCase) error {
	rows, err := q.QueryContext(ctx, `SELECT id,round,reviewer,decision,reason_codes,comment,decided_at,finding_count FROM steward_reviews WHERE case_id=? ORDER BY round`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var review domain.StewardReview
		var reasons, decided string
		review.CaseID = c.ID
		if err := rows.Scan(&review.ID, &review.Round, &review.Reviewer, &review.Decision, &reasons, &review.Comment, &decided, &review.FindingCount); err != nil {
			return err
		}
		if err := decode(reasons, &review.ReasonCodes); err != nil {
			return err
		}
		review.DecidedAt, err = parseTime(decided)
		if err != nil {
			return err
		}
		c.Reviews = append(c.Reviews, review)
	}
	return rows.Err()
}

func (r *SQLiteRepository) ListCases(ctx context.Context, status string) ([]domain.ReleaseCase, error) {
	return r.ListCasesFiltered(ctx, CaseFilter{Status: status})
}

func (r *SQLiteRepository) ListCasesFiltered(ctx context.Context, filter CaseFilter) ([]domain.ReleaseCase, error) {
	query := `SELECT id FROM release_cases`
	args := []any{}
	conditions := []string{}
	if filter.Status != "" {
		conditions = append(conditions, "status=?")
		args = append(args, filter.Status)
	}
	if filter.ReleaseLevel != "" {
		conditions = append(conditions, "release_level=?")
		args = append(args, filter.ReleaseLevel)
	}
	if filter.Owner != "" {
		conditions = append(conditions, "owner=?")
		args = append(args, filter.Owner)
	}
	if filter.CreatedFrom != nil {
		conditions = append(conditions, "created_at>=?")
		args = append(args, filter.CreatedFrom.UTC().Format(timeFormat))
	}
	if filter.CreatedTo != nil {
		conditions = append(conditions, "created_at<=?")
		args = append(args, filter.CreatedTo.UTC().Format(timeFormat))
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY created_at,id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.ReleaseCase, 0, len(ids))
	for _, id := range ids {
		c, err := r.GetCase(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, nil
}

func (r *SQLiteRepository) Timeline(ctx context.Context, caseID string) ([]domain.AuditEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT sequence,actor,request_id,before_revision,after_revision,event_type,occurred_at,details FROM audit_events WHERE case_id=? ORDER BY sequence`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.AuditEvent
	for rows.Next() {
		var e domain.AuditEvent
		var occurred, details string
		e.CaseID = caseID
		if err := rows.Scan(&e.Sequence, &e.Actor, &e.RequestID, &e.BeforeRevision, &e.AfterRevision, &e.EventType, &occurred, &details); err != nil {
			return nil, err
		}
		e.OccurredAt, err = parseTime(occurred)
		if err != nil {
			return nil, err
		}
		if err := decode(details, &e.Details); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *SQLiteRepository) Manifest(ctx context.Context, caseID string) (*domain.ReleaseManifest, error) {
	var m domain.ReleaseManifest
	var assets, constraints, sealed string
	err := r.db.QueryRowContext(ctx, `SELECT id,case_id,case_revision,asset_entries,constraint_summary,canonical_json,sha256,sealed_at FROM release_manifests WHERE case_id=?`, caseID).Scan(&m.ID, &m.CaseID, &m.CaseRevision, &assets, &constraints, &m.CanonicalJSON, &m.SHA256, &sealed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := decode(assets, &m.AssetEntries); err != nil {
		return nil, err
	}
	if err := decode(constraints, &m.ConstraintSummary); err != nil {
		return nil, err
	}
	m.SealedAt, err = parseTime(sealed)
	if err != nil {
		return nil, fmt.Errorf("解析封存时间: %w", err)
	}
	return &m, nil
}
