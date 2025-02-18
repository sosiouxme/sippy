package sippyserver

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	gormlogger "gorm.io/gorm/logger"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"testing"
	"time"

	"github.com/openshift/sippy/pkg/apis/api"
	"github.com/openshift/sippy/pkg/dataloader/prowloader/gcs"
	"github.com/openshift/sippy/pkg/db"
	"github.com/openshift/sippy/pkg/db/models"
)

/*
	  many of these tests require a live database and/or GCS bucket with known data to run,
	  so we do not want them trying to run during CI. We will skip tests whose required environment variables are not set.
	  do not risk checking in credentials, supply them with environment variables:
		TEST_DB_LOG_LEVEL: "silent" or "info" or "warn" or "error" - the log level for gorm database methods
		TEST_SIPPY_DATABASE_DSN: the DSN for the sippy postgres database e.g. postgresql://sippyro:...@sippy-postgresql...amazonaws.com/sippy_openshift
		TEST_GCS_CREDS_PATH: the path to a local GCS credentials file, e.g. /home/$USER/git/sippy/openshift-sippy-ro.creds.json
*/
func dbHandle(t *testing.T) *db.DB {
	dbLogLevel := os.Getenv("TEST_DB_LOG_LEVEL") // e.g. "info" or "silent"
	if dbLogLevel == "" {
		dbLogLevel = "silent"
	}
	gormLogLevel, err := db.ParseGormLogLevel(dbLogLevel)
	if err != nil {
		logrus.WithError(err).Errorf("Cannot parse TEST_DB_LOG_LEVEL %s", dbLogLevel)
		gormLogLevel = gormlogger.Silent
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

func getGcsBucket(t *testing.T) *storage.Client {
	pathToGcsCredentials := os.Getenv("TEST_GCS_CREDS_PATH")
	if pathToGcsCredentials == "" {
		t.Skip("TEST_GCS_CREDS_PATH environment variable is not set; skipping GCS tests")
	}
	gcsClient, err := gcs.NewGCSClient(context.TODO(), pathToGcsCredentials, "")
	if err != nil {
		logrus.WithError(err).Fatal("CRITICAL error getting GCS client which prevents testing")
	}
	return gcsClient
}

func TestMatchPriorRiskAnalysisTest(t *testing.T) {

	tests := map[string]struct {
		priorRiskAnalysisJSON    string
		riskAnalysisJSON         string
		expectedSummaryTestCount int
		expectedRiskLevel        api.RiskLevel
	}{
		"MatchAll": {
			priorRiskAnalysisJSON:    `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684917114550358016,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"[sig-arch] Only known images used by tests","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.04% of 3767 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]},{"Name":"Cluster upgrade.[sig-apps] job-upgrade","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.14% of 1280 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			riskAnalysisJSON:         `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684985307247677440,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"[sig-arch] Only known images used by tests","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.04% of 3767 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]},{"Name":"Cluster upgrade.[sig-apps] job-upgrade","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.14% of 1280 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			expectedSummaryTestCount: 2,
			expectedRiskLevel:        api.FailureRiskLevelHigh,
		},
		"MatchOnePrior": {
			priorRiskAnalysisJSON:    `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684917114550358016,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"[sig-arch] Only known images used by tests","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.04% of 3767 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			riskAnalysisJSON:         `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684985307247677440,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"[sig-arch] Only known images used by tests","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.04% of 3767 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]},{"Name":"Cluster upgrade.[sig-apps] job-upgrade","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.14% of 1280 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			expectedSummaryTestCount: 1,
			expectedRiskLevel:        api.FailureRiskLevelHigh,
		},
		"MatchOneCurrent": {
			priorRiskAnalysisJSON:    `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684917114550358016,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"[sig-arch] Only known images used by tests","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.04% of 3767 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]},{"Name":"Cluster upgrade.[sig-apps] job-upgrade","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.14% of 1280 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			riskAnalysisJSON:         `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684985307247677440,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"Cluster upgrade.[sig-apps] job-upgrade","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.14% of 1280 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			expectedSummaryTestCount: 1,
			expectedRiskLevel:        api.FailureRiskLevelHigh,
		},
		"MatchNone": {
			priorRiskAnalysisJSON:    `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684917114550358016,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"[sig-arch] Only known images used by tests","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.04% of 3767 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			riskAnalysisJSON:         `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684985307247677440,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"Cluster upgrade.[sig-apps] job-upgrade","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.14% of 1280 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			expectedSummaryTestCount: 0,
			expectedRiskLevel:        api.FailureRiskLevelNone,
		},
		"MatchAllHighsNoPrior": {
			riskAnalysisJSON:         `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-upgrade","ProwJobRunID":1684985307247677440,"Release":"Presubmits","CompareRelease":"4.14","Tests":[{"Name":"[sig-arch] Only known images used by tests","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.04% of 3767 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]},{"Name":"[sig-autoscaling] [Feature:HPA] Horizontal pod autoscaling (scale resource:CPU) CustomResourceDefinition Should scale with a CRD targetRef [Suite:openshift/conformance/parallel] [Suite:k8s]","Risk":{"Level":{"Name":"Medium","Level":50},"Reasons":["This test has passed 91.34% of 127 runs on release 4.14 [amd64 aws ha ovn] in the last week."]},"OpenBugs":[]},{"Name":"Cluster upgrade.[sig-apps] job-upgrade","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.14% of 1280 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[]},{"Name":"[bz-kube-apiserver][invariant] alert/KubeAPIErrorBudgetBurn should not be at or above info","Risk":{"Level":{"Name":"High","Level":100},"Reasons":["This test has passed 99.20% of 3764 runs on release 4.14 [Overall] in the last week."]},"OpenBugs":[{"id":15394692,"key":"TRT-1167","created_at":"2023-08-10T02:59:03.979473-04:00","updated_at":"2023-09-14T11:02:57.748888-04:00","deleted_at":null,"status":"In Progress","last_change_time":"2023-09-13T08:35:17-04:00","summary":"Investigate Opportunity For Risk Analysis Tuning","affects_versions":[],"fix_versions":[],"components":[],"labels":[],"url":"https://issues.redhat.com/browse/TRT-1167"}]}],"OverallRisk":{"Level":{"Name":"High","Level":100},"Reasons":["Maximum failed test risk:High"]},"OpenBugs":[]}`,
			expectedSummaryTestCount: 3,
			expectedRiskLevel:        api.FailureRiskLevelHigh,
		},
		"MatchIncompletes": {
			priorRiskAnalysisJSON:    `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-serial","ProwJobRunID":1684917111224274944,"Release":"Presubmits","CompareRelease":"4.14","Tests":[],"OverallRisk":{"Level":{"Name":"IncompleteTests","Level":75},"Reasons":["Tests for this run (100) are below the historical average (709):IncompleteTests"]},"OpenBugs":[]}`,
			riskAnalysisJSON:         `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-serial","ProwJobRunID":1684985307130236928,"Release":"Presubmits","CompareRelease":"4.14","Tests":[],"OverallRisk":{"Level":{"Name":"IncompleteTests","Level":75},"Reasons":["Tests for this run (57) are below the historical average (709):IncompleteTests"]},"OpenBugs":[]}`,
			expectedRiskLevel:        api.FailureRiskLevelIncompleteTests,
			expectedSummaryTestCount: 0,
		},
		"NoMatchIncompletes": {
			priorRiskAnalysisJSON:    `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-serial","ProwJobRunID":1684917111224274944,"Release":"Presubmits","CompareRelease":"4.14","Tests":[],"OverallRisk":{"Level":{"Name":"MissingData","Level":75},"Reasons":["Tests for this run (100) are below the historical average (709):IncompleteTests"]},"OpenBugs":[]}`,
			riskAnalysisJSON:         `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-serial","ProwJobRunID":1684985307130236928,"Release":"Presubmits","CompareRelease":"4.14","Tests":[],"OverallRisk":{"Level":{"Name":"IncompleteTests","Level":75},"Reasons":["Tests for this run (57) are below the historical average (709):IncompleteTests"]},"OpenBugs":[]}`,
			expectedRiskLevel:        api.FailureRiskLevelNone,
			expectedSummaryTestCount: 0,
		},
		"NoMatchIncompletesNoPrior": {
			riskAnalysisJSON:         `{"ProwJobName":"pull-ci-openshift-origin-master-e2e-aws-ovn-serial","ProwJobRunID":1684985307130236928,"Release":"Presubmits","CompareRelease":"4.14","Tests":[],"OverallRisk":{"Level":{"Name":"IncompleteTests","Level":75},"Reasons":["Tests for this run (57) are below the historical average (709):IncompleteTests"]},"OpenBugs":[]}`,
			expectedRiskLevel:        api.FailureRiskLevelNone,
			expectedSummaryTestCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			var priorRiskAnalysis, riskAnalysis api.ProwJobRunRiskAnalysis
			var priorRiskAnalysisPTR *api.ProwJobRunRiskAnalysis

			if len(tc.priorRiskAnalysisJSON) > 0 {
				err := json.Unmarshal([]byte(tc.priorRiskAnalysisJSON), &priorRiskAnalysis)
				assert.Nil(t, err, "Failed to unmarshal prior risk analysis: %v", err)
				priorRiskAnalysisPTR = &priorRiskAnalysis
			}
			err := json.Unmarshal([]byte(tc.riskAnalysisJSON), &riskAnalysis)
			assert.Nil(t, err, "Failed to unmarshal prior risk analysis: %v", err)

			summary := buildRiskSummary(&riskAnalysis, priorRiskAnalysisPTR)

			assert.NotNil(t, summary, "Nil summary")
			assert.Equal(t, tc.expectedSummaryTestCount, len(summary.Tests), "Invalid summary test count")
			assert.Equal(t, tc.expectedRiskLevel, summary.OverallRisk.Level, "Invalid summary risk level")

		})
	}
}

func TestAnalysisWorker(t *testing.T) {
	// initialize AnalysisWorker
	dbc := dbHandle(t)
	gcsClient := getGcsBucket(t)
	logrus.SetLevel(logrus.DebugLevel)

	pendingComments := make(chan PendingComment, 5)
	defer close(pendingComments)
	pendingWork := make(chan models.PullRequestComment, 1)
	defer close(pendingWork)

	analysisWorker := AnalysisWorker{
		riskAnalysisLocator: gcs.GetDefaultRiskAnalysisSummaryFile(),
		dbc:                 dbc,
		gcsBucket:           gcsClient.Bucket("test-platform-results"),
		pendingAnalysis:     pendingWork,
		pendingComments:     pendingComments,
	}

	// prPendingComment := models.PullRequestComment{Org: "openshift", Repo: "origin", PullNumber: 28075, SHA: "79d237196d93eb92ed58c66497d8718259264226", ProwJobRoot: "pr-logs/pull/28075/"}
	prPendingComment := models.PullRequestComment{Org: "openshift", Repo: "origin", PullNumber: 29512, SHA: "8849ed78d4c51e2add729a68a2cbf8551c6d60c9", ProwJobRoot: "pr-logs/pull/29512/"} // mine for testing with one job only
	//prPendingComment := models.PullRequestComment{Org: "openshift", Repo: "origin", PullNumber: 29474, SHA: "58a8615189ebd164a1ce87ffe9b078965a9f4b14", ProwJobRoot: "pr-logs/pull/29474/"}  // currently has a comment
	analysisWorker.processPendingPrComment(prPendingComment)

	pc := <-pendingComments
	logrus.Infof("Pending comment: %+v", pc)
}

func TestBuildJobMap(t *testing.T) {
	// initialize AnalysisWorker
	gcsClient := getGcsBucket(t)
	aw := AnalysisWorker{gcsBucket: gcsClient.Bucket("test-platform-results")}
	logrus.SetLevel(logrus.DebugLevel)
	logger := logrus.WithContext(context.TODO())
	//logrus.Infof("saw job map %v", aw.buildProwJobRuns(logger, "pr-logs/pull/29512/pull-ci-openshift-origin-master-e2e-aws-ovn-single-node/"))
	logrus.Infof("saw job map %v", aw.buildProwJobRuns(logger, "pr-logs/pull/29501/pull-ci-openshift-origin-master-e2e-aws-ovn-edge-zones/"))
}

func TestIsNewTest(t *testing.T) {
	dbc := dbHandle(t)

	ntf := &pgNewTestFilter{
		dbc:         dbc,
		notNewTests: sets.Set[uint]{},
	}
	logrus.SetLevel(logrus.DebugLevel)
	logger := logrus.WithContext(context.TODO())

	test := models.Test{Name: "[sig-sippy] openshift-tests should work"}
	test.ID = 522
	test.CreatedAt = time.Now()
	isNew, err := ntf.IsNewTest(logger, test)

	assert.Nil(t, err, "Failed to check if test is new")
	assert.Equal(t, false, isNew, "Test should not be new")

	test.Name = "a failed test that has never been seen before"
	test.ID = 160471
	isNew, err = ntf.IsNewTest(logger, test)

	assert.Nil(t, err, "Failed to check if test is new")
	assert.Equal(t, true, isNew, "Test should be new")
}
