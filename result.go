package emailkit

// Result is the full outcome of an email validation.
// The Valid field is true only if all configured checks passed.
type Result struct {
	Email  string        `json:"email"`
	Valid  bool          `json:"valid"`
	Checks []CheckResult `json:"checks"`
}

// FailedChecks returns those CheckResults that did not pass.
func (r Result) FailedChecks() []CheckResult {
	var out []CheckResult
	for _, c := range r.Checks {
		if !c.Passed {
			out = append(out, c)
		}
	}
	return out
}

// CheckFor returns the CheckResult for the given level, if it exists.
// The second return value indicates whether the given level was executed.
func (r Result) CheckFor(level CheckLevel) (CheckResult, bool) {
	for _, c := range r.Checks {
		if c.Level == level {
			return c, true
		}
	}
	return CheckResult{}, false
}
