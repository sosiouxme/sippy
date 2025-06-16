package componentreport

import (
	"time"

	crtier1 "github.com/openshift/sippy/pkg/apis/api/componentreport/test"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/testdetails"
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
	// TODO: really feels like this could just be moved  TestComparison, eliminating the need for ReportTestSummary
	crtier1.ReportTestIdentification
	testdetails.TestComparison
}
