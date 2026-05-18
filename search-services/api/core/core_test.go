package core

import "testing"

func TestSentinelErrors(t *testing.T) {
	if ErrBadArguments == nil || ErrAlreadyExists == nil || ErrNotFound == nil {
		t.Fatalf("expected sentinel errors to be initialized")
	}
}

func TestUpdateStatusValues(t *testing.T) {
	if StatusUpdateUnknown == "" || StatusUpdateIdle == StatusUpdateRunning {
		t.Fatalf("unexpected status constants")
	}
}
