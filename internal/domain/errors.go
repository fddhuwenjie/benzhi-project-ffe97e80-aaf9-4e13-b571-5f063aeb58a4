package domain

import "fmt"

type RuleError struct {
	Code    string
	Message string
}

func (e *RuleError) Error() string { return e.Message }

func NewRuleError(code, message string) error {
	return &RuleError{Code: code, Message: message}
}

func Required(field string) error {
	return NewRuleError("validation_error", fmt.Sprintf("%s 不能为空", field))
}
