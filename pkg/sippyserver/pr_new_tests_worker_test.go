package sippyserver

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"github.com/openshift/sippy/pkg/api"
	"github.com/openshift/sippy/pkg/apis/prow"
	"github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
	"github.com/openshift/sippy/pkg/dataloader/prowloader/gcs"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/db/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/*
	  many of these are functional tests requiring a live database and/or GCS bucket with known data to run,
	  so we do not want them trying to run during CI. We will skip tests whose required environment variables are not set.
	  don't risk checking in credentials with code; supply them with environment variables:
		TEST_DB_LOG_LEVEL: "silent" or "info" or "warn" or "error" - the log level for gorm database methods
		TEST_SIPPY_DATABASE_DSN: the DSN for the sippy postgres database e.g. postgresql://sippyro:...@sippy-postgresql...amazonaws.com/sippy_openshift
		TEST_GCS_CREDS_PATH: the path to a local GCS credentials file, e.g. /home/$USER/git/sippy/openshift-sippy-ro.creds.json
*/
func getDbHandle(t *testing.T) *db.DB {
	dbLogLevel := os.Getenv("TEST_DB_LOG_LEVEL") // e.g. "info" or "silent"
	if dbLogLevel == "" {
		dbLogLevel = "silent"
	}
	gormLogLevel, err := db.ParseGormLogLevel(dbLogLevel)
	if err != nil {
		logrus.WithError(err).Errorf("Cannot parse TEST_DB_LOG_LEVEL %s", dbLogLevel)
		gormLogLevel = logger.Silent
	}

	dsn := os.Getenv("TEST_SIPPY_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_SIPPY_DATABASE_DSN environment variable is not set; skipping database tests")
	}
	dbc, err := db.New(dsn, gormLogLevel)
	if err != nil {
		logrus.WithError(err).Fatal("Cannot connect to database")
	}
	return dbc
}

func getGcsBucket(t *testing.T) *storage.BucketHandle {
	pathToGcsCredentials := os.Getenv("TEST_GCS_CREDS_PATH")
	if pathToGcsCredentials == "" {
		t.Skip("TEST_GCS_CREDS_PATH environment variable is not set; skipping GCS tests")
	}
	gcsClient, err := gcs.NewGCSClient(context.TODO(), pathToGcsCredentials, "")
	if err != nil {
		logrus.WithError(err).Fatalf("CRITICAL error getting GCS client with credentials at %s", pathToGcsCredentials)
	}
	return gcsClient.Bucket("test-platform-results")
}

func TestBuildJobMap(t *testing.T) {
	// initialize AnalysisWorker
	gcsBucket := getGcsBucket(t)
	aw := AnalysisWorker{gcsBucket: gcsBucket}
	logrus.SetLevel(logrus.DebugLevel)
	logger := logrus.WithContext(context.TODO())
	//logrus.Infof("saw job map %v", aw.buildProwJobRuns(logger, "pr-logs/pull/29512/pull-ci-openshift-origin-master-e2e-aws-ovn-single-node/"))
	logrus.Infof("saw job map %v", aw.buildProwJobRuns(logger, "pr-logs/pull/29501/pull-ci-openshift-origin-master-e2e-aws-ovn-edge-zones/"))
}

func TestAssessJobRisks(t *testing.T) {
	logger := logrus.WithContext(context.TODO())
	logrus.SetLevel(logrus.DebugLevel)

	// Initialize a standard NewTestsWorker
	dbc := getDbHandle(t)
	ntf := &pgNewTestFilter{dbc: dbc, notNewTests: sets.Set[uint]{}}
	ntw := &NewTestsWorker{dbc: dbc, newTestFilter: ntf, fetchJobRun: api.FetchJobRun}

	// Initialize GCS client and look up known job in the bucket
	bucket := getGcsBucket(t)
	aw := AnalysisWorker{gcsBucket: bucket, newTestsWorker: ntw}
	jobRuns := aw.buildProwJobRuns(logger, "pr-logs/pull/29512/pull-ci-openshift-origin-master-e2e-aws-ovn-single-node/")
	if !assert.True(t, len(jobRuns) > 0, "Failed to load job runs") {
		return // expected to use the first job run as a test subject
	}
	if !assert.Equal(t, "1885131315280351232", jobRuns[0].Status.BuildID, "Unexpected build ID") {
		return
	}

	// Assess job risks
	jobRisks := ntw.assessJobRisks(logger, []*prow.ProwJob{&jobRuns[0]})
	if !assert.Equalf(t, len(jobRisks), 2, "expect risks only for the two that were new; saw %+v", jobRisks) {
		return
	}
	failed, ok := jobRisks["a failed test that has never been seen before"]
	if assert.True(t, ok, "Should have found failed test") {
		assert.Equal(t, 1, failed.Failures, "Unexpected number of failures")
	}
	passed, ok := jobRisks["a passed test that has never been seen before"]
	if assert.True(t, ok, "Should have found failed test") {
		assert.Equal(t, 0, passed.Failures, "Unexpected failure found")
	}
}

func TestAssessMultiRunRisks(t *testing.T) {
	// TODO
}

func newTest(name string, success, failure bool) NewTest {
	return NewTest{
		TestName: name,
		Success:  success,
		Failure:  failure,
	}
}

func TestRiskScenarios(t *testing.T) {
	cases := []struct {
		name     string
		tests    []NewTest
		expected NewTestRisk
	}{ // all assume two job runs
		{
			name: "AllTestsPassing",
			tests: []NewTest{
				newTest("test", true, false),
				newTest("test", true, false),
			},
			expected: NewTestRisk{
				Failures:   0,
				Flakes:     0,
				AnyMissing: false,
			},
		},
		{
			name: "SomeTestsFailing",
			tests: []NewTest{
				newTest("test", true, false),
				newTest("test", false, true),
			},
			expected: NewTestRisk{
				Failures:   1,
				Flakes:     0,
				AnyMissing: false,
			},
		},
		{
			name: "FlakyTest and MissingTest",
			tests: []NewTest{
				newTest("test", true, true),
			},
			expected: NewTestRisk{
				Failures:   0,
				Flakes:     1,
				AnyMissing: true,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			risk := makeNewTestRisk("test", 2, c.tests)
			assert.Equal(t, c.expected.Failures, risk.Failures)
			assert.Equal(t, c.expected.Flakes, risk.Flakes)
			assert.Equal(t, c.expected.AnyMissing, risk.AnyMissing)
		})
	}
}

func TestUnit_getNewTestsForJobRun(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	jobRun := &prow.ProwJob{
		Spec:   prow.ProwJobSpec{Job: "test-jobRun"},
		Status: prow.ProwJobStatus{BuildID: "12345"},
	}
	tests := []struct {
		name          string
		fetchJobRun   func(dbc *db.DB, jobRunID int64, unknownTests bool, logger *logrus.Entry) (*models.ProwJobRun, int, error)
		testFilter    NewTestFilter
		expectedTests []NewTest
		expectedError error
	}{
		{
			name: "successful fetch",
			fetchJobRun: func(dbc *db.DB, jobRunID int64, unknownTests bool, logger *logrus.Entry) (*models.ProwJobRun, int, error) {
				pjr := models.ProwJobRun{
					Tests: []models.ProwJobRunTest{
						{Test: models.Test{Name: "test1"}, Status: int(v1.TestStatusSuccess)},
						{Test: models.Test{Name: "test2"}, Status: int(v1.TestStatusFailure)},
						{Test: models.Test{Name: "test3"}, Status: int(v1.TestStatusFlake)},
					},
				}
				pjr.ID = 12345 // Gorm model ID for some reason can't be put in the struct literal
				return &pjr, 0, nil
			},
			testFilter: &oneNewTestFilter{}, // filters to only "test2"
			expectedTests: []NewTest{
				{JobName: "test-jobRun", JobRunID: 12345, TestName: "test2", Success: false, Failure: true},
			},
			expectedError: nil,
		},
		{
			name: "error on filtering",
			fetchJobRun: func(dbc *db.DB, jobRunID int64, unknownTests bool, logger *logrus.Entry) (*models.ProwJobRun, int, error) {
				pjr := models.ProwJobRun{
					Tests: []models.ProwJobRunTest{
						{Test: models.Test{Name: "test1"}, Status: int(v1.TestStatusSuccess)},
					},
				}
				pjr.ID = 12345 // Gorm model ID for some reason can't be put in the struct literal
				return &pjr, 0, nil
			},
			testFilter:    &errorNewTestFilter{}, // mocks a failure in the filter
			expectedTests: nil,
			expectedError: errors.New("filter error"),
		},
		{
			name: "jobRun run not found",
			fetchJobRun: func(dbc *db.DB, jobRunID int64, unknownTests bool, logger *logrus.Entry) (*models.ProwJobRun, int, error) {
				return nil, 0, gorm.ErrRecordNotFound
			},
			expectedTests: nil,
			expectedError: gorm.ErrRecordNotFound,
		},
		{
			name: "error fetching jobRun run",
			fetchJobRun: func(dbc *db.DB, jobRunID int64, unknownTests bool, logger *logrus.Entry) (*models.ProwJobRun, int, error) {
				return nil, 0, errors.New("fetch error")
			},
			expectedTests: nil,
			expectedError: errors.New("fetch error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ntw := &NewTestsWorker{
				dbc:           nil,
				newTestFilter: tt.testFilter,
				fetchJobRun:   tt.fetchJobRun,
			}
			newTests, err := ntw.getNewTestsForJobRun(logger, jobRun)
			assert.Equal(t, tt.expectedTests, newTests)
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

type oneNewTestFilter struct{}
type errorNewTestFilter struct{}

func (m *oneNewTestFilter) IsNewTest(logger *logrus.Entry, test models.ProwJobRunTest) (bool, error) {
	if test.Test.Name == "test2" {
		return true, nil
	}
	return false, nil
}
func (m *errorNewTestFilter) IsNewTest(logger *logrus.Entry, test models.ProwJobRunTest) (bool, error) {
	return false, errors.New("filter error")
}

func TestFunc_getNewTestsForJobRun(t *testing.T) {
	dbc := getDbHandle(t)

	// Initialize a standard NewTestsWorker
	ntf := &pgNewTestFilter{dbc: dbc, notNewTests: sets.Set[uint]{}}
	ntw := &NewTestsWorker{dbc: dbc, newTestFilter: ntf, fetchJobRun: api.FetchJobRun}

	// try with a known job run
	jobRun := &prow.ProwJob{
		Spec:   prow.ProwJobSpec{Job: "pull-ci-openshift-origin-master-e2e-aws-ovn-single-node"},
		Status: prow.ProwJobStatus{BuildID: "1885131315280351232"},
	}
	logrus.SetLevel(logrus.DebugLevel)
	logger := logrus.WithContext(context.TODO())
	newTests, err := ntw.getNewTestsForJobRun(logger, jobRun)

	fmt.Printf("new tests: %v\n", newTests)
	assert.NoError(t, err, "Failed to get new tests")
	assert.Equal(t, 2, len(newTests), "Unexpected number of new tests")
	assert.True(t, ntf.notNewTests.Has(522), "Test 522 should be considered not new")
	assert.False(t, ntf.notNewTests.Has(160471), "Test 160471 should be left out to be considered new")
	assert.False(t, ntf.notNewTests.Has(160472), "Test 160472 should be left out to be considered new")
}

func TestIsNewTest(t *testing.T) {
	dbc := getDbHandle(t)

	ntf := &pgNewTestFilter{
		dbc:         dbc,
		notNewTests: sets.Set[uint]{},
	}
	logrus.SetLevel(logrus.DebugLevel)
	logger := logrus.WithContext(context.TODO())

	test := models.Test{Name: "[sig-sippy] openshift-tests should work"}
	test.ID = 522
	testRun := models.ProwJobRunTest{
		Test:      test,
		TestID:    test.ID,
		CreatedAt: time.Now(),
	}
	isNew, err := ntf.IsNewTest(logger, testRun)

	assert.Nil(t, err, "Failed to check if test is new")
	assert.Equal(t, false, isNew, "Test should not be new")

	test.Name = "a failed test that has never been seen before"
	test.ID = 160471
	testRun.Test = test
	testRun.TestID = test.ID
	isNew, err = ntf.IsNewTest(logger, testRun)

	assert.Nil(t, err, "Failed to check if test is new")
	assert.Equal(t, true, isNew, "Test should be new")
}
