package domain

import "encoding/json"

func CloneCase(source *ReleaseCase) (*ReleaseCase, error) {
	data, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var result ReleaseCase
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func CloneEvents(source []AuditEvent) ([]AuditEvent, error) {
	data, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var result []AuditEvent
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
