package gatus

import "testing"

func TestApplyGuardedDNS(t *testing.T) {
	t.Parallel()
	t.Run("populates fields", func(t *testing.T) {
		t.Parallel()
		e := &Endpoint{}
		ApplyGuardedDNS("example.com", e)
		if e.URL != GuardedProbeURL {
			t.Errorf("URL = %q, want %q", e.URL, GuardedProbeURL)
		}
		if e.DNS["query-name"] != "example.com" || e.DNS["query-type"] != GuardedQueryType {
			t.Errorf("DNS = %v", e.DNS)
		}
		if len(e.Conditions) != 1 || e.Conditions[0] != GuardedEmptyBodyCondition {
			t.Errorf("Conditions = %v", e.Conditions)
		}
	})

	t.Run("empty host is no-op", func(t *testing.T) {
		t.Parallel()
		e := &Endpoint{}
		ApplyGuardedDNS("", e)
		if e.URL != "" || e.DNS != nil || e.Conditions != nil {
			t.Errorf("ApplyGuardedDNS with empty host should not mutate: %+v", e)
		}
	})

	t.Run("nil endpoint is no-op", func(t *testing.T) {
		t.Parallel()
		// just verify it doesn't panic
		ApplyGuardedDNS("example.com", nil)
	})
}
