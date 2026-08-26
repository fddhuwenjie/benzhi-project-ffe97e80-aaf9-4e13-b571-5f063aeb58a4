package domain

import (
	"strings"
	"testing"
	"time"
)

func TestReleaseWorkflowAndReturn(t *testing.T) {
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	c := &ReleaseCase{ID: "case-1", LanguageName: "测试方言", CollectionBatch: "B-1", Owner: "管理员", ReleaseLevel: "PUBLIC", Status: StatusDraft, Revision: 1, CreatedAt: now}
	grant := ConsentGrant{ID: "consent-1", ParticipantCode: "P1", EvidenceRef: "evidence://1", PermittedUses: []string{"PUBLIC"}}
	if err := c.AddConsent(grant, now); err != nil {
		t.Fatal(err)
	}
	asset := RecordingAsset{ID: "asset-1", StableKey: "audio-001", Summary: "访谈材料", DurationMS: 10000, CapturedOn: "2026-08-20", ParticipantCodes: []string{"P1"}, ContentSHA256: strings.Repeat("a", 64)}
	if err := c.AddAsset(asset, now); err != nil {
		t.Fatal(err)
	}
	finding := SensitivityFinding{ID: "finding-1", StartMS: 10, EndMS: 50, Category: "CULTURAL", Severity: "HIGH", Disposition: "MUTE", TreatmentNote: "已完成静音脱敏处置", Status: "OPEN"}
	if err := c.AddFinding("asset-1", finding); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitForReview(now); err == nil {
		t.Fatal("存在未关闭发现时不应送审")
	}
	if err := c.CloseFinding("finding-1", "复核确认静音处置完整"); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitForReview(now); err != nil {
		t.Fatal(err)
	}
	if err := c.Review(StewardReview{ID: "review-1", Reviewer: "社区代表", Decision: "RETURN", ReasonCodes: []string{"DETAIL"}, Comment: "需要补充说明", DecidedAt: now}); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusRedactionReview || len(c.Reviews) != 1 {
		t.Fatalf("退回状态或历史错误: %+v", c)
	}
	if err := c.SubmitForReview(now); err != nil {
		t.Fatal(err)
	}
	if err := c.Review(StewardReview{ID: "review-2", Reviewer: "社区代表", Decision: "APPROVE", DecidedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := c.Seal(now); err != nil {
		t.Fatal(err)
	}
	if err := c.AddFinding("asset-1", finding); err == nil {
		t.Fatal("封存后修改应失败")
	}
}

func TestConsentCoverageAndWithdrawal(t *testing.T) {
	now := time.Now().UTC()
	c := &ReleaseCase{ID: "c", Status: StatusDraft, ReleaseLevel: "PUBLIC"}
	if err := c.AddConsent(ConsentGrant{ID: "g", ParticipantCode: "P", EvidenceRef: "ref", PermittedUses: []string{"RESTRICTED"}}, now); err != nil {
		t.Fatal(err)
	}
	asset := RecordingAsset{ID: "a", StableKey: "key-001", Summary: "摘要", DurationMS: 1, CapturedOn: "2026-08-26", ParticipantCodes: []string{"P"}, ContentSHA256: strings.Repeat("b", 64)}
	if err := c.AddAsset(asset, now); err == nil {
		t.Fatal("用途不覆盖时不应接受材料")
	}
	if err := c.WithdrawConsent("g", now); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusDraft || c.Consents[0].WithdrawnAt == nil {
		t.Fatal("撤回状态未记录")
	}
}

func TestExpiredConsentBlocksSubmission(t *testing.T) {
	now := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)
	c := &ReleaseCase{ID: "c", Status: StatusDraft, ReleaseLevel: "PUBLIC"}
	if err := c.AddConsent(ConsentGrant{ID: "g", ParticipantCode: "P", EvidenceRef: "ref", PermittedUses: []string{"PUBLIC"}, ValidUntil: &expires}, now); err != nil {
		t.Fatal(err)
	}
	asset := RecordingAsset{ID: "a", StableKey: "key-001", Summary: "摘要", DurationMS: 1, CapturedOn: "2026-08-26", ParticipantCodes: []string{"P"}, ContentSHA256: strings.Repeat("b", 64)}
	if err := c.AddAsset(asset, now); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitForReview(now.Add(2 * time.Hour)); err == nil {
		t.Fatal("授权过期后不应允许送审")
	}
}
