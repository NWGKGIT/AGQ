package state

import (
	"reflect"
	"testing"
	"time"

	"agq-daemon/internal/domain"
)

func TestSnapshotCopiesEmailsAndTimes(t *testing.T) {
	st := New()
	emails := []string{"one@example.com", "two@example.com"}
	st.SetActive(emails)

	last := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	next := last.Add(time.Minute)
	st.SetPollTimes(last, next)

	snapshot := st.Snapshot()
	if snapshot.State != domain.StateActive {
		t.Fatalf("State = %s, want ACTIVE", snapshot.State)
	}
	if !reflect.DeepEqual(snapshot.Emails, emails) {
		t.Fatalf("Emails = %#v, want %#v", snapshot.Emails, emails)
	}

	emails[0] = "mutated@example.com"
	snapshot.Emails[1] = "mutated@example.com"
	second := st.Snapshot()
	if !reflect.DeepEqual(second.Emails, []string{"one@example.com", "two@example.com"}) {
		t.Fatalf("state emails were aliased: %#v", second.Emails)
	}

	*snapshot.LastPollAt = snapshot.LastPollAt.Add(time.Hour)
	third := st.Snapshot()
	if !third.LastPollAt.Equal(last) {
		t.Fatalf("LastPollAt was aliased: %s", third.LastPollAt)
	}
}

func TestSetIdleClearsActiveData(t *testing.T) {
	st := New()
	now := time.Now()
	st.SetActive([]string{"user@example.com"})
	st.SetPollTimes(now, now.Add(time.Minute))

	st.SetIdle()
	snapshot := st.Snapshot()
	if snapshot.State != domain.StateIdle {
		t.Fatalf("State = %s, want IDLE", snapshot.State)
	}
	if len(snapshot.Emails) != 0 {
		t.Fatalf("Emails = %#v, want empty", snapshot.Emails)
	}
	if snapshot.NextPollAt != nil {
		t.Fatalf("NextPollAt = %v, want nil", snapshot.NextPollAt)
	}
}
