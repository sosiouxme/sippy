package test

import (
	"encoding/json"
	"time"
)

type ColumnID string

type ColumnIdentification struct {
	Variants map[string]string `json:"variants"`
}

type RowIdentification struct {
	Component  string `json:"component"`
	Capability string `json:"capability,omitempty"`
	TestName   string `json:"test_name,omitempty"`
	TestSuite  string `json:"test_suite,omitempty"`
	TestID     string `json:"test_id,omitempty"`
}

type ReportTestIdentification struct {
	RowIdentification
	ColumnIdentification
}

// Comparison is the type of comparison done for a test that has been marked red.
type Comparison string

type Status int

type Release struct {
	Release string
	End     *time.Time
	Start   *time.Time
}

func StringForStatus(s Status) string {
	switch s {
	case ExtremeRegression:
		return "Extreme"
	case SignificantRegression:
		return "Significant"
	case ExtremeTriagedRegression:
		return "ExtremeTriaged"
	case SignificantTriagedRegression:
		return "SignificantTriaged"
	case MissingSample:
		return "MissingSample"
	case FixedRegression:
		return "Fixed"
	case FailedFixedRegression:
		return "FailedFixed"
	}
	return "Unknown"
}

// JobVariants contains all variants supported in the system.
type JobVariants struct {
	Variants map[string][]string `json:"variants,omitempty"`
}

// KeyWithVariants connects the core unique db testID string to a set of variants.
// Used to serialize/deserialize as a map key when we pass test status around.
type KeyWithVariants struct {
	TestID string `json:"test_id"`

	// Proposed, need to serialize to use as map key
	Variants map[string]string `json:"variants"`
}

// KeyOrDie serializes this test key into a json string suitable for use in maps.
// JSON serialization uses sorted map keys, so the output is stable.
func (t KeyWithVariants) KeyOrDie() string {
	testIDBytes, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return string(testIDBytes)
}
