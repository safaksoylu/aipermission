package api

import "fmt"

const (
	maxCommandBytes = 64 << 10
	maxReasonBytes  = 2 << 10
	maxMessageBytes = 8 << 10
)

func validateTextLimit(field string, value string, limit int) error {
	if len([]byte(value)) <= limit {
		return nil
	}
	return fmt.Errorf("%s must be %d bytes or less", field, limit)
}
