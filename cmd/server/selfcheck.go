package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type smokeClient struct {
	base   string
	client *http.Client
}

func runSelfCheck(address string) error {
	c := smokeClient{base: "http://" + address, client: &http.Client{Timeout: 5 * time.Second}}
	if err := c.waitReady(); err != nil {
		return err
	}
	created, err := c.request("POST", "/api/v1/cases", map[string]any{"request_id": "self-create", "actor": "资料管理员", "language_name": "示例方言", "collection_batch": "SELF-CHECK-01", "owner": "自检负责人", "release_level": "PUBLIC"}, 201)
	if err != nil {
		return err
	}
	caseID := nestedString(created, "case", "id")
	if caseID == "" {
		return fmt.Errorf("建档响应缺少 case.id")
	}
	revision := int64(1)
	steps := []struct {
		method, path string
		body         map[string]any
		status       int
	}{
		{"POST", "/api/v1/cases/" + caseID + "/consents", command(revision, "self-consent", map[string]any{"participant_code": "P001", "evidence_ref": "consent://self/P001", "permitted_uses": []string{"PUBLIC"}, "region_limits": []string{"CN"}}), 200},
		{"POST", "/api/v1/cases/" + caseID + "/assets", command(revision+1, "self-asset", map[string]any{"stable_key": "SELF-AUDIO-001", "summary": "自检录音材料摘要", "duration_ms": 60000, "captured_on": "2026-08-26", "participant_codes": []string{"P001"}, "content_sha256": strings.Repeat("a", 64)}), 200},
	}
	var assetID string
	for _, step := range steps {
		value, err := c.request(step.method, step.path, step.body, step.status)
		if err != nil {
			return err
		}
		revision = int64(value["revision"].(float64))
		if step.path[len(step.path)-6:] == "assets" {
			assetID = nestedString(value, "case", "assets", "0", "id")
		}
	}
	if assetID == "" {
		return fmt.Errorf("材料响应缺少 asset id")
	}
	value, err := c.request("POST", "/api/v1/cases/"+caseID+"/assets/"+assetID+"/findings", command(revision, "self-finding-1", map[string]any{"start_ms": 1000, "end_ms": 2000, "category": "CULTURAL", "severity": "HIGH", "disposition": "MUTE", "treatment_note": "已按社区规则完成静音处置", "status": "CLOSED"}), 200)
	if err != nil {
		return err
	}
	revision = revisionOf(value)
	value, err = c.request("POST", "/api/v1/cases/"+caseID+"/submit", command(revision, "self-submit-1", nil), 200)
	if err != nil {
		return err
	}
	revision = revisionOf(value)
	value, err = c.request("POST", "/api/v1/cases/"+caseID+"/reviews", command(revision, "self-return", map[string]any{"reviewer": "社区代表甲", "decision": "RETURN", "reason_codes": []string{"MORE_REDACTION"}, "comment": "需要补充身份信息脱敏"}), 200)
	if err != nil {
		return err
	}
	revision = revisionOf(value)
	value, err = c.request("POST", "/api/v1/cases/"+caseID+"/assets/"+assetID+"/findings", command(revision, "self-finding-2", map[string]any{"start_ms": 3000, "end_ms": 4000, "category": "IDENTITY", "severity": "MEDIUM", "disposition": "VOICE_SHIFT", "treatment_note": "已完成说话人变声脱敏处置", "status": "CLOSED"}), 200)
	if err != nil {
		return err
	}
	revision = revisionOf(value)
	value, err = c.request("POST", "/api/v1/cases/"+caseID+"/submit", command(revision, "self-submit-2", nil), 200)
	if err != nil {
		return err
	}
	revision = revisionOf(value)
	value, err = c.request("POST", "/api/v1/cases/"+caseID+"/reviews", command(revision, "self-approve-review", map[string]any{"reviewer": "社区代表甲", "decision": "APPROVE", "reason_codes": []string{}, "comment": "脱敏结果符合社区发布要求"}), 200)
	if err != nil {
		return err
	}
	revision = revisionOf(value)
	value, err = c.request("POST", "/api/v1/cases/"+caseID+"/approve", command(revision, "self-seal", nil), 200)
	if err != nil {
		return err
	}
	revision = revisionOf(value)
	if nestedString(value, "manifest", "sha256") == "" {
		return fmt.Errorf("封存响应缺少摘要")
	}
	if _, err = c.request("GET", "/api/v1/cases/"+caseID+"/timeline", nil, 200); err != nil {
		return err
	}
	if _, err = c.request("GET", "/api/v1/cases/"+caseID+"/manifest", nil, 200); err != nil {
		return err
	}
	_, err = c.request("POST", "/api/v1/cases/"+caseID+"/submit", command(revision, "self-after-seal", nil), 409)
	return err
}

func command(revision int64, requestID string, extra map[string]any) map[string]any {
	body := map[string]any{"request_id": requestID, "expected_revision": revision, "actor": "自检执行者"}
	for key, value := range extra {
		body[key] = value
	}
	return body
}
func revisionOf(value map[string]any) int64 { return int64(value["revision"].(float64)) }

func nestedString(value map[string]any, path ...string) string {
	var current any = value
	for _, part := range path {
		switch node := current.(type) {
		case map[string]any:
			current = node[part]
		case []any:
			if part != "0" || len(node) == 0 {
				return ""
			}
			current = node[0]
		default:
			return ""
		}
	}
	result, _ := current.(string)
	return result
}

func (c smokeClient) waitReady() error {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response, err := c.client.Get(c.base + "/readyz")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("服务就绪探测超时")
}

func (c smokeClient) request(method, path string, body map[string]any, want int) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != want {
		return nil, fmt.Errorf("%s %s 返回 %d，期望 %d: %s", method, path, response.StatusCode, want, data)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("解析 %s 响应: %w", path, err)
	}
	return decoded, nil
}
