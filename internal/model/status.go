package model

import "fmt"

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusClosed     Status = "closed"
)

var validStatuses = []Status{StatusOpen, StatusInProgress, StatusClosed}

func ValidateStatus(s Status) error {
	for _, v := range validStatuses {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf("invalid status %q: must be one of open, in_progress, closed", s)
}
