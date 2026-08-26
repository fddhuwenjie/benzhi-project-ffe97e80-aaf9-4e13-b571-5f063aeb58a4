package audit

import (
	"dialect-release/internal/domain"
	"time"
)

func NewEvent(caseID, actor, requestID, eventType string, before, after int64, at time.Time, details map[string]any) domain.AuditEvent {
	return domain.AuditEvent{CaseID: caseID, Actor: actor, RequestID: requestID, BeforeRevision: before, AfterRevision: after, EventType: eventType, OccurredAt: at.UTC(), Details: details}
}

func VerifyTimeline(events []domain.AuditEvent) error {
	var previous int64
	for index, event := range events {
		if index == 0 {
			if event.BeforeRevision != 0 || event.AfterRevision != 1 {
				return domain.NewRuleError("audit_revision_gap", "审计时间线起始修订不完整")
			}
		} else if event.BeforeRevision != previous || event.AfterRevision != previous+1 {
			return domain.NewRuleError("audit_revision_gap", "审计时间线存在修订断层")
		}
		previous = event.AfterRevision
	}
	return nil
}
