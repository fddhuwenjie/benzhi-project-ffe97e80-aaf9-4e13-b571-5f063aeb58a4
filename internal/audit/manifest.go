package audit

import (
	"crypto/sha256"
	"dialect-release/internal/domain"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

type canonicalManifest struct {
	SchemaVersion string                 `json:"schema_version"`
	CaseID        string                 `json:"case_id"`
	CaseRevision  int64                  `json:"case_revision"`
	Assets        []domain.ManifestAsset `json:"assets"`
	Constraints   []string               `json:"constraints"`
}

func BuildManifest(id string, c *domain.ReleaseCase, sealedAt time.Time) (*domain.ReleaseManifest, error) {
	if err := c.ValidateAuthorizationCoverage(sealedAt); err != nil {
		return nil, err
	}
	assets, constraints, err := c.ManifestInput()
	if err != nil {
		return nil, err
	}
	payload := canonicalManifest{SchemaVersion: "1", CaseID: c.ID, CaseRevision: c.Revision + 1, Assets: assets, Constraints: constraints}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("编码规范清单: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return &domain.ReleaseManifest{ID: id, CaseID: c.ID, CaseRevision: payload.CaseRevision, AssetEntries: assets, ConstraintSummary: constraints, CanonicalJSON: string(encoded), SHA256: hex.EncodeToString(sum[:]), SealedAt: sealedAt.UTC()}, nil
}

func VerifyManifest(manifest *domain.ReleaseManifest) error {
	sum := sha256.Sum256([]byte(manifest.CanonicalJSON))
	if hex.EncodeToString(sum[:]) != manifest.SHA256 {
		return domain.NewRuleError("manifest_digest_mismatch", "封存清单摘要校验失败")
	}
	var decoded canonicalManifest
	if err := json.Unmarshal([]byte(manifest.CanonicalJSON), &decoded); err != nil {
		return domain.NewRuleError("manifest_invalid", "封存清单规范 JSON 无效")
	}
	if decoded.CaseID != manifest.CaseID || decoded.CaseRevision != manifest.CaseRevision {
		return domain.NewRuleError("manifest_metadata_mismatch", "封存清单元数据不一致")
	}
	if !reflect.DeepEqual(decoded.Assets, manifest.AssetEntries) || !reflect.DeepEqual(decoded.Constraints, manifest.ConstraintSummary) {
		return domain.NewRuleError("manifest_metadata_mismatch", "封存清单内容与规范 JSON 不一致")
	}
	return nil
}
