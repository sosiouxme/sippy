package sippyserver

import (
	"errors"
	"fmt"
	"github.com/openshift/sippy/pkg/db/models"
	"gorm.io/gorm"
	"strconv"

	jobQueries "github.com/openshift/sippy/pkg/api"
	"github.com/openshift/sippy/pkg/apis/api"
	"github.com/openshift/sippy/pkg/apis/prow"
	spv1 "github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
	"github.com/openshift/sippy/pkg/db"
	"github.com/sirupsen/logrus"
)

type NewTest struct {
	JobName  string
	JobRunID uint
	TestName string
	Success  bool
	Failure  bool // and if both, it's a flake
}

// NewTestRisk is created for the following scenarios:
// 1. Any PR job adds a new test that appears in one run and not another at the same sha - high risk
// 2. PR adds new test that appears in only a single job and:
//   - it fails at all - high risk
//   - it succeeds or flakes - medium risk (might not be intended for multiple jobs)
//
// 3. PR adds new test that appears in more than one job and (at latest sha):
//   - it fails at all  - high risk
//   - it succeeds or flakes - no risk (only included in list of all new tests)
type NewTestRisk struct {
	TestName   string
	AnyMissing bool // it was a new test in one run but missing in another at same sha
	Runs       int  // how many job runs did we examine
	Failures   int  // how many of the test results were failure
	Flakes     int  // or flakes
	OnlyInOne  bool // new test was only seen in one job of multiple for this PR
	NewTests   []NewTest
	Level      api.RiskLevel
	Reason     string
}

type JobNewTestRisks struct {
	JobName      string
	NewTestRisks map[string]NewTestRisk // one risk record per new test name
}

type NewTestFilter interface {
	IsNewTest(testName string) (bool, error)
}

// BQNewTestFilter uses bigquery to determine if a test is new. We can share
// a single instance between workers and cache results so we are not constantly
// querying BQ for the same test.
type BQNewTestFilter struct {
	// we will have some form of cache and a bigquery client at least.
}

type NewTestsWorker struct {
	dbc           *db.DB
	newTestFilter NewTestFilter
}

// analyzeRisks walks the runs for a PR job looking for new tests and assessing their risk
func (ntw *NewTestsWorker) analyzeRisks(logger *logrus.Entry, jobs []prJobInfo) (jobRisks []JobNewTestRisks) {
	for _, jobInfo := range jobs {
		latestRuns := ntw.filterJobRunsForNewTests(logger, jobInfo)
		if latestRuns == nil {
			logger.Infof(
				"Skipping new test analysis for job %s as there are no completed runs against the PR's shasum %s",
				jobInfo.name, jobInfo.prShaSum)
			continue
		}

		risks := ntw.assessJobRisks(logger, latestRuns)
		if risks != nil {
			// now we have the risks for this job, we need to be able to merge them into the overall analysis
			jobRisks = append(jobRisks, JobNewTestRisks{JobName: jobInfo.name, NewTestRisks: risks})
		}
	}

	if len(jobRisks) > 0 {
		// look across the PR's jobs and upgrade risks for new tests that are only found in one job
		assessCrossJobRisks(jobRisks, jobs)
		// and finally, assign risk levels given everything we know about the new tests
		assignRiskLevels(jobRisks)
	}

	return
}

// look across the PR's jobs and upgrade risks for new tests that are only found in one job
func assessCrossJobRisks(jobRisks []JobNewTestRisks, jobs []prJobInfo) {
	if len(jobs) < 2 {
		return // we need at least two jobs to compare new tests
	}

	// first figure out how many jobs saw each new test
	newTestJobCount := make(map[string]int)
	for _, jobRisk := range jobRisks {
		for testName := range jobRisk.NewTestRisks {
			newTestJobCount[testName]++
		}
	}

	// upgrade risk of any new test that is unique to one job
	for _, jobRisk := range jobRisks {
		for testName, risk := range jobRisk.NewTestRisks {
			if newTestJobCount[testName] == 1 {
				risk.OnlyInOne = true
			}
		}
	}
}

func assignRiskLevels(jobRisks []JobNewTestRisks) {
	for _, jobRisk := range jobRisks {
		for _, risk := range jobRisk.NewTestRisks {
			if risk.AnyMissing {
				// 1. Any PR job adds a new test that appears in one run and not another at the same sha - high risk
				if risk.Failures > 0 {
					//   - it fails at all - high risk
					risk.Level = api.FailureRiskLevelHigh
					risk.Reason = fmt.Sprintf("is a new test that was not present in all runs against the current commit, and also failed %d time(s).", risk.Failures)
				} else {
					//   - it succeeds or flakes - medium risk (might not be intended for multiple jobs)
					risk.Level = api.FailureRiskLevelHigh
					risk.Reason = "is a new test, and was only seen in one job."
				}

			} else if risk.OnlyInOne {
				// 2. PR adds new test that appears in only a single job and:
				if risk.Failures > 0 {
					//   - it fails at all - high risk
					risk.Level = api.FailureRiskLevelHigh
					risk.Reason = fmt.Sprintf("is a new test, was only seen in one job, and failed %d time(s) against the current commit.", risk.Failures)
				} else {
					//   - it succeeds or flakes - medium risk (might not be intended for multiple jobs)
					risk.Level = api.FailureRiskLevelMedium
					risk.Reason = "is a new test, and was only seen in one job."
				}
			} else {
				// 3. PR adds new test that appears in more than one job and (at latest sha):
				if risk.Failures > 0 {
					//   - it fails at all - high risk
					risk.Level = api.FailureRiskLevelHigh
					risk.Reason = fmt.Sprintf("is a new test that failed %d time(s) against the current commit", risk.Failures)
				} else {
					//   - it succeeds or flakes - no risk (only included in list of all new tests)
					risk.Level = api.FailureRiskLevelNone
				}
			}
		}
	}
}

func (ntw *NewTestsWorker) filterJobRunsForNewTests(logger *logrus.Entry, jobInfo prJobInfo) []*prow.ProwJob {
	// we need up to 2 runs that ran against the PR's shasum (do not flag problems from different shasums)
	var latestRuns []*prow.ProwJob
	for idx, run := range jobInfo.prowJobRuns {
		if run.Spec.Refs.Pulls[0].SHA != jobInfo.prShaSum {
			continue
		}
		if ntw.isIncompleteRun(run) {
			continue
		}
		latestRuns = append(latestRuns, &jobInfo.prowJobRuns[idx])
	}
	return latestRuns
}

// filter out incomplete runs (with significantly fewer tests than usual -- not related to ProwJob status);
// when looking for new tests, runs that didn't get to all the tests will muddy the analysis.
// such runs should be left to risk analysis for comment.
func (ntw *NewTestsWorker) isIncompleteRun(run prow.ProwJob) bool {
	// TODO
	return false
}

func (ntw *NewTestsWorker) assessJobRisks(logger *logrus.Entry, jobRuns []*prow.ProwJob) map[string]NewTestRisk {
	// find the new tests in all the comparable runs we have for one job
	newTestsByName := map[string][]NewTest{}
	for _, run := range jobRuns {
		if newTests, err := ntw.getNewTestsForJobRun(logger, run); err == nil {
			for _, test := range newTests {
				newTestsByName[test.TestName] = append(newTestsByName[test.TestName], test)
			}
		}
	}

	// evaluate each test for risks
	risksByName := make(map[string]NewTestRisk, 0)
	for testName, tests := range newTestsByName {
		// new tests in general are a low risk if they succeed, mainly we want to record their existence
		risk := NewTestRisk{
			TestName:   testName,
			AnyMissing: false,
			Runs:       len(jobRuns),
			NewTests:   tests,
		}

		// new tests that fail in any runs are a high risk
		for _, test := range tests {
			if test.Failure && test.Success {
				// sometimes new tests are deliberately introduced to flake;
				// count these for informational purposes but do not consider them an extra risk
				risk.Flakes += 1
			} else if test.Failure {
				risk.Failures += 1
			}
		}

		// with multiple runs, check whether new tests also showed up in all runs;
		// if not, they likely either have dynamic names or do not consistently run, either of which is a risk
		if len(tests) < len(jobRuns) {
			risk.AnyMissing = true
		}

		risksByName[testName] = risk
		// later at the PR level, we can further compare new tests across multiple jobs;
		// a new test that is only seen in one job is a risk similar to one not seen across all runs.
	}
	return risksByName
}

func (ntw *NewTestsWorker) getNewTestsForJobRun(logger *logrus.Entry, prowjob *prow.ProwJob) (newTests []NewTest, err error) {
	logger = logger.WithField("func", "getNewTestsForJobRun").WithField("job", prowjob.Spec.Job).WithField("run", prowjob.Status.BuildID)
	var jobRun *models.ProwJobRun
	if jobRunIntID, err := strconv.ParseInt(prowjob.Status.BuildID, 10, 64); err != nil {
		// this would be exceedingly strange
		logger.WithError(err).Error("Failed to parse jobRunId id")
		return nil, err
	} else if jobRun, _, err = jobQueries.FetchJobRun(ntw.dbc, jobRunIntID, true, logger); err != nil {
		// RecordNotFound can be expected if the jobRunId job isn't in sippy yet. log any other error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Debug("Job run not found")
		} else {
			logger.WithError(err).Error("Error fetching job run")
		}
		return nil, err
	}
	for _, test := range jobRun.Tests {
		if new, err := ntw.newTestFilter.IsNewTest(test.Test.Name); err != nil {
			logger.WithError(err).Error("Error checking if test is new")
			return nil, err // if this breaks, it muddies this job's analysis
		} else if new {
			newTests = append(newTests, NewTest{
				JobName:  prowjob.Spec.Job,
				JobRunID: jobRun.ID,
				TestName: test.Test.Name,
				Success:  test.Status == int(spv1.TestStatusSuccess) || test.Status == int(spv1.TestStatusFlake),
				Failure:  test.Status == int(spv1.TestStatusFailure) || test.Status == int(spv1.TestStatusFlake),
			})
		}
	}
	return newTests, nil
}

func (*BQNewTestFilter) IsNewTest(testName string) (bool, error) {
	// TODO
	return false, nil
}
