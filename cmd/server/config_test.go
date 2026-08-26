package main

import "testing"

func TestAddressValidation(t *testing.T) {
	for _, valid := range []string{"127.0.0.1:19081", "localhost:20000", "[::1]:21000"} {
		if err := validateAddress(valid); err != nil {
			t.Errorf("%s 应有效: %v", valid, err)
		}
	}
	for _, invalid := range []string{"0.0.0.0:19081", "127.0.0.1", "127.0.0.1:99999"} {
		if err := validateAddress(invalid); err == nil {
			t.Errorf("%s 应无效", invalid)
		}
	}
}
