package application

import (
	"context"
	"dialect-release/internal/audit"
	"dialect-release/internal/domain"
	"dialect-release/internal/store"
	"encoding/json"
	"time"
)

func (s *Service) GetCase(ctx context.Context, id string) (*domain.ReleaseCase, error) {
	return s.repo.GetCase(ctx, id)
}
func (s *Service) ListCases(ctx context.Context, status string) ([]domain.ReleaseCase, error) {
	return s.repo.ListCases(ctx, status)
}

func (s *Service) ListCasesQuery(ctx context.Context, query CaseListQuery) (CaseListResult, error) {
	filter := store.CaseFilter{Status: query.Status, ReleaseLevel: query.ReleaseLevel, Owner: query.Owner}
	var err error
	if query.CreatedFrom != "" {
		value, e := time.Parse("2006-01-02", query.CreatedFrom)
		if e != nil {
			return CaseListResult{}, domain.NewRuleError("validation_error", "created_from 日期无效")
		}
		filter.CreatedFrom = &value
	}
	if query.CreatedTo != "" {
		value, e := time.Parse("2006-01-02", query.CreatedTo)
		if e != nil {
			return CaseListResult{}, domain.NewRuleError("validation_error", "created_to 日期无效")
		}
		value = value.Add(24*time.Hour - time.Nanosecond)
		filter.CreatedTo = &value
	}
	if query.Status != "" && !domain.CaseStatus(query.Status).Valid() {
		return CaseListResult{}, domain.NewRuleError("validation_error", "status 无效")
	}
	if query.ReleaseLevel != "" && query.ReleaseLevel != "PUBLIC" && query.ReleaseLevel != "RESTRICTED" && query.ReleaseLevel != "COMMUNITY_ONLY" {
		return CaseListResult{}, domain.NewRuleError("validation_error", "release_level 无效")
	}
	var values []domain.ReleaseCase
	if filtered, ok := s.repo.(store.FilteredRepository); ok {
		values, err = filtered.ListCasesFiltered(ctx, filter)
	} else {
		values, err = s.repo.ListCases(ctx, query.Status)
	}
	if err != nil {
		return CaseListResult{}, err
	}
	counts := map[string]int{}
	for _, c := range values {
		counts[string(c.Status)]++
	}
	if counts == nil {
		counts = map[string]int{}
	}
	return CaseListResult{Cases: values, Count: len(values), StatusCounts: counts}, nil
}
func (s *Service) Timeline(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	return s.TimelinePage(ctx, id, TimelineQuery{})
}

func (s *Service) TimelinePage(ctx context.Context, id string, query TimelineQuery) ([]domain.AuditEvent, error) {
	if _, err := s.repo.GetCase(ctx, id); err != nil {
		return nil, err
	}
	events, err := s.repo.Timeline(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := audit.VerifyTimeline(events); err != nil {
		return nil, err
	}
	if query.AfterRevision > 0 || query.BeforeRevision > 0 {
		filtered := events[:0]
		for _, event := range events {
			if query.AfterRevision > 0 && event.AfterRevision <= query.AfterRevision {
				continue
			}
			if query.BeforeRevision > 0 && event.AfterRevision >= query.BeforeRevision {
				continue
			}
			filtered = append(filtered, event)
		}
		events = filtered
	}
	if query.Limit > 0 && len(events) > query.Limit {
		events = events[:query.Limit]
	}
	return events, nil
}
func (s *Service) Manifest(ctx context.Context, id string) (*domain.ReleaseManifest, error) {
	if encoded, ok := s.manifestCache[id]; ok {
		var cached domain.ReleaseManifest
		if err := json.Unmarshal(encoded, &cached); err != nil {
			return nil, err
		}
		return &cached, nil
	}
	manifest, err := s.repo.Manifest(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := audit.VerifyManifest(manifest); err != nil {
		return nil, err
	}
	caseValue, err := s.repo.GetCase(ctx, id)
	if err != nil {
		return nil, err
	}
	if caseValue.Status != domain.StatusSealed || caseValue.Revision != manifest.CaseRevision {
		return nil, domain.NewRuleError("manifest_metadata_mismatch", "封存清单与个案修订不一致")
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	s.manifestCache[id] = encoded
	return manifest, nil
}
