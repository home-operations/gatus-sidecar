package gatus

// Guarded probes replace a direct HTTP check with a DNS query to a public
// resolver (Cloudflare). Used when the sidecar pod can't reach the service
// directly but DNS resolution is still meaningful.
const (
	GuardedProbeURL           = "1.1.1.1"
	GuardedQueryType          = "A"
	GuardedEmptyBodyCondition = "len([BODY]) == 0"
)

// ApplyGuardedDNS rewrites e in place to perform a DNS lookup of host.
func ApplyGuardedDNS(host string, e *Endpoint) {
	if host == "" || e == nil {
		return
	}
	e.URL = GuardedProbeURL
	e.DNS = map[string]any{
		"query-name": host,
		"query-type": GuardedQueryType,
	}
	e.Conditions = []string{GuardedEmptyBodyCondition}
}
