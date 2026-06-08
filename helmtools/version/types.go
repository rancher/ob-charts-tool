package version

// TagMismatch describes a single discrepancy between a rule's expected tag value and what was
// found in a values.yaml map.
type TagMismatch struct {
	ValuesKey     string
	ActualValue   string
	ExpectedValue string
}
