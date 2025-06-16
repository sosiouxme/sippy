package bq

import (
	"math/big"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/test"
)

// TestStatus is an internal type used to pass data bigquery onwards to the actual
// report generation. It is not serialized over the API.
type TestStatus struct {
	TestName     string   `json:"test_name"`
	TestSuite    string   `json:"test_suite"`
	Component    string   `json:"component"`
	Capabilities []string `json:"capabilities"`
	Variants     []string `json:"variants"`
	test.Count
	LastFailure time.Time `json:"last_failure"`
}

// ReportTestStatus contains the mapping of all test keys (serialized with KeyWithVariants, variants + testID)
// It is also an internal type used to pass data from bigquery onwards to report generation, and does not get serialized
// as an API response.
type ReportTestStatus struct {
	// BaseStatus represents the stable basis for the comparison. Maps KeyWithVariants serialized as a string, to test status.
	BaseStatus map[string]TestStatus `json:"base_status"`

	// SampleSatus represents the sample for the comparison. Maps KeyWithVariants serialized as a string, to test status.
	SampleStatus map[string]TestStatus `json:"sample_status"`
	GeneratedAt  *time.Time            `json:"generated_at"`
}

type Variant struct {
	Key   string `bigquery:"key" json:"key"`
	Value string `bigquery:"value" json:"value"`
}

// TODO: temporary for migration
type TestRegressionBigQuery struct {
	// Snapshot is the time at which the full set of regressions for all releases was inserted into the db.
	// When querying we use only those with the latest snapshot time.
	Snapshot     time.Time              `bigquery:"snapshot" json:"snapshot"`
	View         string                 `bigquery:"view" json:"view"`
	Release      string                 `bigquery:"release" json:"release"`
	TestID       string                 `bigquery:"test_id" json:"test_id"`
	TestName     string                 `bigquery:"test_name" json:"test_name"`
	RegressionID string                 `bigquery:"regression_id" json:"regression_id"`
	Opened       time.Time              `bigquery:"opened" json:"opened"`
	Closed       bigquery.NullTimestamp `bigquery:"closed" json:"closed"`
	Variants     []Variant              `bigquery:"variants" json:"variants"`
}

// JobVariant defines a variant and the possible values
type JobVariant struct {
	VariantName   string   `bigquery:"variant_name"`
	VariantValues []string `bigquery:"variant_values"`
}

// TestJobRunRows are the per job run rows that come back from bigquery for a test details report
// indicating if the test passed or failed.
// Fields are named count somewhat misleadingly as technically they're always 0 or 1 today.
type TestJobRunRows struct {
	TestKey      test.KeyWithVariants `json:"test_key"`
	TestKeyStr   string               `json:"-"` // transient field so we dont have to keep recalculating
	TestName     string               `bigquery:"test_name"`
	ProwJob      string               `bigquery:"prowjob_name"`
	ProwJobRunID string               `bigquery:"prowjob_run_id"`
	ProwJobURL   string               `bigquery:"prowjob_url"`
	StartTime    civil.DateTime       `bigquery:"prowjob_start"`
	test.Count
	JiraComponent   string   `bigquery:"jira_component"`
	JiraComponentID *big.Rat `bigquery:"jira_component_id"`
}
