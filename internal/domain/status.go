package domain

import "fmt"

type CaseStatus string

const (
	StatusDraft              CaseStatus = "DRAFT"
	StatusConsentReady       CaseStatus = "CONSENT_READY"
	StatusMaterialRegistered CaseStatus = "MATERIAL_REGISTERED"
	StatusRedactionReview    CaseStatus = "REDACTION_REVIEW"
	StatusStewardReview      CaseStatus = "STEWARD_REVIEW"
	StatusApproved           CaseStatus = "APPROVED"
	StatusSealed             CaseStatus = "SEALED"
)

func (s CaseStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusConsentReady, StatusMaterialRegistered, StatusRedactionReview, StatusStewardReview, StatusApproved, StatusSealed:
		return true
	default:
		return false
	}
}

func requireStatus(actual CaseStatus, allowed ...CaseStatus) error {
	for _, candidate := range allowed {
		if actual == candidate {
			return nil
		}
	}
	return NewRuleError("invalid_status", fmt.Sprintf("当前状态 %s 不允许执行此操作", actual))
}
