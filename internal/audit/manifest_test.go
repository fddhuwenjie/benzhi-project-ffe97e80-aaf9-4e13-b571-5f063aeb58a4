package audit

import (
	"dialect-release/internal/domain"
	"strings"
	"testing"
	"time"
)

func TestManifestIsDeterministicAndVerified(t *testing.T) {
	now := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	c := &domain.ReleaseCase{ID: "case-1", Status: domain.StatusApproved, Revision: 8, ReleaseLevel: "PUBLIC", Assets: []domain.RecordingAsset{
		{StableKey: "z-key", Summary: "Z", DurationMS: 2, CapturedOn: "2026-01-02", ParticipantCodes: []string{"P2", "P1"}, ContentSHA256: strings.Repeat("a", 64)},
		{StableKey: "a-key", Summary: "A", DurationMS: 1, CapturedOn: "2026-01-01", ParticipantCodes: []string{"P1"}, ContentSHA256: strings.Repeat("b", 64)},
	}, Consents: []domain.ConsentGrant{
		{ParticipantCode: "P1", PermittedUses: []string{"PUBLIC"}, RegionLimits: []string{"CN", "LOCAL"}},
		{ParticipantCode: "P2", PermittedUses: []string{"PUBLIC"}, RegionLimits: []string{"CN"}},
	}}
	first, err := BuildManifest("m1", c, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildManifest("m2", c, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.CanonicalJSON != second.CanonicalJSON || first.SHA256 != second.SHA256 {
		t.Fatal("同一输入必须产生相同清单")
	}
	if first.AssetEntries[0].StableKey != "a-key" {
		t.Fatal("材料未按 stable_key 排序")
	}
	if err := VerifyManifest(first); err != nil {
		t.Fatal(err)
	}
	first.CanonicalJSON += " "
	if err := VerifyManifest(first); err == nil {
		t.Fatal("篡改后的清单应校验失败")
	}
}
