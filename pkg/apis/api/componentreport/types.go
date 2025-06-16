package componentreport

import (
	"encoding/json"
	"math/big"
	"time"

	"cloud.google.com/go/civil"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/bq"
	crtier1 "github.com/openshift/sippy/pkg/apis/api/componentreport/tier1"
	"github.com/openshift/sippy/pkg/db/models"
)

type ComponentReport struct {
	Rows        []ReportRow `json:"rows,omitempty"`
	GeneratedAt *time.Time  `json:"generated_at"`
}

type ReportRow struct {
	crtier1.RowIdentification
	Columns []ReportColumn `json:"columns,omitempty"`
}

type ReportColumn struct {
	crtier1.ColumnIdentification
	Status         crtier1.Status      `json:"status"`
	RegressedTests []ReportTestSummary `json:"regressed_tests,omitempty"`
}

type ReportTestSummary struct {
	// TODO: really feels like this could just be moved  ReportTestStats, eliminating the need for ReportTestSummary
	crtier1.ReportTestIdentification
	ReportTestStats
}

// ReportTestStats is an overview struct for a particular regressed test's stats.
// (basis passes and pass rate, sample passes and pass rate, and fishers exact confidence)
// Important type returned by the API.
// TODO: compare with TestStatus we use internally, see if we can converge?
type ReportTestStats struct {
	// ReportStatus is an integer representing the severity of the regression.
	ReportStatus crtier1.Status `json:"status"`

	// Comparison indicates what mode was used to check this tests results in the sample.
	Comparison crtier1.Comparison `json:"comparison"`

	// Explanations are human-readable details of why this test was marked regressed.
	Explanations []string `json:"explanations"`

	SampleStats TestDetailsReleaseStats `json:"sample_stats"`

	// RequiredConfidence is the confidence required from Fishers to consider a regression.
	// Typically, it is as defined in the request options, but middleware may choose to adjust.
	// 95 = 95% confidence of a regression required.
	RequiredConfidence int `json:"-"`

	// PityAdjustment can be used to adjust the tolerance for failures for this particular test.
	PityAdjustment float64 `json:"-"`

	// RequiredPassRateAdjustment can be used to adjust the tolerance for failures for a new test.
	RequiredPassRateAdjustment float64 `json:"-"`

	// Optional fields depending on the Comparison mode

	// FisherExact indicates the confidence of a regression after applying Fisher's Exact Test.
	FisherExact *float64 `json:"fisher_exact,omitempty"`

	// BaseStats may not be present in the response, i.e. new tests regressed because of their pass rate.
	BaseStats *TestDetailsReleaseStats `json:"base_stats,omitempty"`

	// LastFailure is the last time the regressed test failed.
	LastFailure *time.Time `json:"last_failure"`

	// Regression is populated with data on when we first detected this regression. If unset it implies
	// the regression tracker has not yet run to find it, or you're using report params/a view without regression tracking.
	Regression *models.TestRegression `json:"regression,omitempty"`
}

// ReportTestDetails is the top level API response for test details reports.
type ReportTestDetails struct {
	crtier1.ReportTestIdentification
	JiraComponent   string     `json:"jira_component"`
	JiraComponentID *big.Rat   `json:"jira_component_id"`
	TestName        string     `json:"test_name"`
	GeneratedAt     *time.Time `json:"generated_at"`

	// Analyses is a list of potentially multiple analysis run for this test.
	// Callers can assume that the first in the list is somewhat authoritative, and should
	// be displayed by default, but each analysis offers details and explanations on it's outcome
	// and can be used in some capacity.
	Analyses []TestDetailsAnalysis `json:"analyses"`
}

// TestDetailsAnalysis is a collection of stats for the report which could potentially carry
// multiple different analyses run.
type TestDetailsAnalysis struct {
	ReportTestStats
	JobStats []TestDetailsJobStats `json:"job_stats,omitempty"`
}

type TestDetailsReleaseStats struct {
	Release string `json:"release"`
	Start   *time.Time
	End     *time.Time
	TestDetailsTestStats
}

type TestDetailsTestStats struct {
	SuccessCount int `json:"success_count"`
	FailureCount int `json:"failure_count"`
	FlakeCount   int `json:"flake_count"`
	// calculate from the above with PassRate method:
	SuccessRate float64 `json:"success_rate"`
}

func (tdts TestDetailsTestStats) Total() int {
	return tdts.SuccessCount + tdts.FailureCount + tdts.FlakeCount
}

func (tdts TestDetailsTestStats) Passes(flakesAsFailure bool) int {
	if flakesAsFailure {
		return tdts.SuccessCount
	}
	return tdts.SuccessCount + tdts.FlakeCount
}

func (tdts TestDetailsTestStats) PassRate(flakesAsFailure bool) float64 {
	return CalculatePassRate(tdts.SuccessCount, tdts.FailureCount, tdts.FlakeCount, flakesAsFailure)
}

func (tdts TestDetailsTestStats) Add(add TestDetailsTestStats, flakesAsFailure bool) TestDetailsTestStats {
	return NewTestStats(
		tdts.SuccessCount+add.SuccessCount,
		tdts.FailureCount+add.FailureCount,
		tdts.FlakeCount+add.FlakeCount,
		flakesAsFailure,
	)
}

func (tdts TestDetailsTestStats) AddTestCount(add bq.TestCount, flakesAsFailure bool) TestDetailsTestStats {
	return NewTestStats(
		tdts.SuccessCount+add.SuccessCount,
		tdts.FailureCount+add.Failures(),
		tdts.FlakeCount+add.FlakeCount,
		flakesAsFailure,
	)
}

func (tdts TestDetailsTestStats) FailPassWithFlakes(flakesAsFailure bool) (int, int) {
	if flakesAsFailure {
		return tdts.FailureCount + tdts.FlakeCount, tdts.SuccessCount
	}
	return tdts.FailureCount, tdts.SuccessCount + tdts.FlakeCount
}

func NewTestStats(successCount, failureCount, flakeCount int, flakesAsFailure bool) TestDetailsTestStats {
	return TestDetailsTestStats{
		SuccessCount: successCount,
		FailureCount: failureCount,
		FlakeCount:   flakeCount,
		SuccessRate:  CalculatePassRate(successCount, failureCount, flakeCount, flakesAsFailure),
	}
}

func CalculatePassRate(success, failure, flake int, treatFlakeAsFailure bool) float64 {
	total := success + failure + flake
	if total == 0 {
		return 0.0
	}
	if treatFlakeAsFailure {
		return float64(success) / float64(total)
	}
	return float64(success+flake) / float64(total)
}

type TestDetailsJobStats struct {
	// one of sample/base job name could be missing if jobs change between releases
	SampleJobName     string                   `json:"sample_job_name,omitempty"`
	BaseJobName       string                   `json:"base_job_name,omitempty"`
	SampleStats       TestDetailsTestStats     `json:"sample_stats"`
	BaseStats         TestDetailsTestStats     `json:"base_stats"`
	SampleJobRunStats []TestDetailsJobRunStats `json:"sample_job_run_stats,omitempty"`
	BaseJobRunStats   []TestDetailsJobRunStats `json:"base_job_run_stats,omitempty"`
	Significant       bool                     `json:"significant"`
}

type TestDetailsJobRunStats struct {
	JobURL    string         `json:"job_url"`
	JobRunID  string         `json:"job_run_id"`
	StartTime civil.DateTime `json:"start_time"`
	// TestStats is the test stats from one particular job run.
	// For the majority of the tests, there is only one junit. But
	// there are cases multiple junits are generated for the same test.
	TestStats TestDetailsTestStats `json:"test_stats"`
}

// TestJobRunStatuses contains the rows returned from a test details query organized by base and sample,
// essentially the actual job runs and their status that was used to calculate this
// report.
// Status fields map prowjob name to each row result we received for that job.
type TestJobRunStatuses struct {
	BaseStatus map[string][]bq.TestJobRunRows `json:"base_status"`
	// TODO: This could be a little cleaner if we did status.BaseStatuses plural and tied them to a release,
	// allowing the release fallback mechanism to stay a little cleaner. That would more clearly
	// keep middleware details out of the main codebase.
	BaseOverrideStatus map[string][]bq.TestJobRunRows `json:"base_override_status"`
	SampleStatus       map[string][]bq.TestJobRunRows `json:"sample_status"`
	GeneratedAt        *time.Time                     `json:"generated_at"`
}

type TestVariants struct {
	Network  []string `json:"network,omitempty"`
	Upgrade  []string `json:"upgrade,omitempty"`
	Arch     []string `json:"arch,omitempty"`
	Platform []string `json:"platform,omitempty"`
	Variant  []string `json:"variant,omitempty"`
}

// JobVariants contains all variants supported in the system.
type JobVariants struct {
	Variants map[string][]string `json:"variants,omitempty"`
}

// TestWithVariantsKey connects the core unique db testID string to a set of variants.
// Used to serialize/deserialize as a map key when we pass test status around.
type TestWithVariantsKey struct {
	TestID string `json:"test_id"`

	// Proposed, need to serialize to use as map key
	Variants map[string]string `json:"variants"`
}

// KeyOrDie serializes this test key into a json string suitable for use in maps.
// JSON serialization uses sorted map keys, so the output is stable.
func (t TestWithVariantsKey) KeyOrDie() string {
	testIDBytes, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return string(testIDBytes)
}
