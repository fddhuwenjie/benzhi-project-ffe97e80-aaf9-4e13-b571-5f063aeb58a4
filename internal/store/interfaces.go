package store

import (
	"context"
	"dialect-release/internal/domain"
	"time"
)

type IdempotencyRecord struct {
	RequestID  string
	CaseID     string
	Operation  string
	StatusCode int
	Body       []byte
}

type Tx interface {
	LoadCase(context.Context, string) (*domain.ReleaseCase, error)
	InsertCase(context.Context, *domain.ReleaseCase) error
	SaveCase(context.Context, *domain.ReleaseCase, int64) error
	AppendEvent(context.Context, domain.AuditEvent) error
	SaveManifest(context.Context, *domain.ReleaseManifest) error
	GetIdempotency(context.Context, string) (*IdempotencyRecord, error)
	SaveIdempotency(context.Context, IdempotencyRecord) error
	Commit() error
	Rollback() error
}

type Repository interface {
	Begin(context.Context) (Tx, error)
	GetCase(context.Context, string) (*domain.ReleaseCase, error)
	ListCases(context.Context, string) ([]domain.ReleaseCase, error)
	Timeline(context.Context, string) ([]domain.AuditEvent, error)
	Manifest(context.Context, string) (*domain.ReleaseManifest, error)
	Close() error
}

type FilteredRepository interface {
	ListCasesFiltered(context.Context, CaseFilter) ([]domain.ReleaseCase, error)
}

type CaseFilter struct {
	Status       string
	ReleaseLevel string
	Owner        string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
}
