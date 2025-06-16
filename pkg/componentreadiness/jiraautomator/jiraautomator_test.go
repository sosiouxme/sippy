package jiraautomator

import (
	"fmt"
	"testing"

	crtype "github.com/openshift/sippy/pkg/apis/api/componentreport"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/test"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/testdetails"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/tier1"
	jiratype "github.com/openshift/sippy/pkg/apis/jira/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetComponentRegressedTestsFromReport(t *testing.T) {
	columnAWSAMD64OVN := tier1.ColumnIdentification{
		Variants: map[string]string{
			"Platform":     "aws",
			"Architecture": "amd64",
			"Network":      "ovn",
		},
	}
	columnAzureAMD64OVN := tier1.ColumnIdentification{
		Variants: map[string]string{
			"Platform":     "aws",
			"Architecture": "amd64",
			"Network":      "ovn",
		},
	}
	columnMetalAMD64OVN := tier1.ColumnIdentification{
		Variants: map[string]string{
			"Platform":     "metal",
			"Architecture": "amd64",
			"Network":      "ovn",
		},
	}
	awsAMD64OVNTest := test.KeyWithVariants{
		TestID: "1",
		Variants: map[string]string{
			"Platform":     "aws",
			"Architecture": "amd64",
			"Network":      "ovn",
		},
	}
	testName1 := "Test 1"
	testName2 := "Test 2"

	tests := []struct {
		name           string
		report         crtype.ComponentReport
		expectedResult map[JiraComponent][]crtype.ReportTestSummary
	}{
		{
			name: "component to regressed tests by component only",
			report: crtype.ComponentReport{
				Rows: []crtype.ReportRow{
					{
						RowIdentification: tier1.RowIdentification{
							Component: "component 1",
						},
						Columns: []crtype.ReportColumn{
							{
								ColumnIdentification: columnAWSAMD64OVN,
								Status:               tier1.ExtremeRegression,
								RegressedTests: []crtype.ReportTestSummary{
									{
										ReportTestIdentification: tier1.ReportTestIdentification{
											RowIdentification: tier1.RowIdentification{
												TestName: testName1,
											},
											ColumnIdentification: tier1.ColumnIdentification{
												Variants: awsAMD64OVNTest.Variants,
											},
										},
										ReportTestStats: testdetails.ReportTestStats{
											ReportStatus: tier1.ExtremeRegression,
										},
									},
								},
							},
							{
								ColumnIdentification: columnAzureAMD64OVN,
								Status:               tier1.ExtremeRegression,
								RegressedTests: []crtype.ReportTestSummary{
									{
										ReportTestIdentification: tier1.ReportTestIdentification{
											RowIdentification: tier1.RowIdentification{
												TestName: testName1,
											},
											ColumnIdentification: tier1.ColumnIdentification{
												Variants: columnAzureAMD64OVN.Variants,
											},
										},
										ReportTestStats: testdetails.ReportTestStats{
											ReportStatus: tier1.ExtremeRegression,
										},
									},
								},
							},
						},
					},
					{
						RowIdentification: tier1.RowIdentification{
							Component: "component 2",
						},
						Columns: []crtype.ReportColumn{
							{
								ColumnIdentification: columnAWSAMD64OVN,
								Status:               tier1.NotSignificant,
								RegressedTests:       []crtype.ReportTestSummary{},
							},
							{
								ColumnIdentification: columnAzureAMD64OVN,
								Status:               tier1.ExtremeRegression,
								RegressedTests: []crtype.ReportTestSummary{
									{
										ReportTestIdentification: tier1.ReportTestIdentification{
											RowIdentification: tier1.RowIdentification{
												TestName: testName2,
											},
											ColumnIdentification: tier1.ColumnIdentification{
												Variants: columnAzureAMD64OVN.Variants,
											},
										},
										ReportTestStats: testdetails.ReportTestStats{
											ReportStatus: tier1.ExtremeRegression,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedResult: map[JiraComponent][]crtype.ReportTestSummary{
				{Project: "OCPBUGS", Component: "component 1"}: {
					{
						ReportTestIdentification: tier1.ReportTestIdentification{
							RowIdentification: tier1.RowIdentification{
								TestName: testName1,
							},
							ColumnIdentification: tier1.ColumnIdentification{
								Variants: awsAMD64OVNTest.Variants,
							},
						},
						ReportTestStats: testdetails.ReportTestStats{
							ReportStatus: tier1.ExtremeRegression,
						},
					},
					{
						ReportTestIdentification: tier1.ReportTestIdentification{
							RowIdentification: tier1.RowIdentification{
								TestName: testName1,
							},
							ColumnIdentification: tier1.ColumnIdentification{
								Variants: columnAzureAMD64OVN.Variants,
							},
						},
						ReportTestStats: testdetails.ReportTestStats{
							ReportStatus: tier1.ExtremeRegression,
						},
					},
				},
				{Project: "OCPBUGS", Component: "component 2"}: {
					{
						ReportTestIdentification: tier1.ReportTestIdentification{
							RowIdentification: tier1.RowIdentification{
								TestName: testName2,
							},
							ColumnIdentification: tier1.ColumnIdentification{
								Variants: columnAzureAMD64OVN.Variants,
							},
						},
						ReportTestStats: testdetails.ReportTestStats{
							ReportStatus: tier1.ExtremeRegression,
						},
					},
				},
			},
		},
		{
			name: "component to regressed tests by component and column grouping",
			report: crtype.ComponentReport{
				Rows: []crtype.ReportRow{
					{
						RowIdentification: tier1.RowIdentification{
							Component: "component 1",
						},
						Columns: []crtype.ReportColumn{
							{
								ColumnIdentification: columnAWSAMD64OVN,
								Status:               tier1.ExtremeRegression,
								RegressedTests: []crtype.ReportTestSummary{
									{
										ReportTestIdentification: tier1.ReportTestIdentification{
											RowIdentification: tier1.RowIdentification{
												TestName: testName1,
											},
											ColumnIdentification: tier1.ColumnIdentification{
												Variants: awsAMD64OVNTest.Variants,
											},
										},
										ReportTestStats: testdetails.ReportTestStats{
											ReportStatus: tier1.ExtremeRegression,
										},
									},
								},
							},
							{
								ColumnIdentification: columnMetalAMD64OVN,
								Status:               tier1.ExtremeRegression,
								RegressedTests: []crtype.ReportTestSummary{
									{
										ReportTestIdentification: tier1.ReportTestIdentification{
											RowIdentification: tier1.RowIdentification{
												TestName: testName1,
											},
											ColumnIdentification: tier1.ColumnIdentification{
												Variants: columnMetalAMD64OVN.Variants,
											},
										},
										ReportTestStats: testdetails.ReportTestStats{
											ReportStatus: tier1.ExtremeRegression,
										},
									},
								},
							},
						},
					},
					{
						RowIdentification: tier1.RowIdentification{
							Component: "component 2",
						},
						Columns: []crtype.ReportColumn{
							{
								ColumnIdentification: columnAWSAMD64OVN,
								Status:               tier1.NotSignificant,
								RegressedTests:       []crtype.ReportTestSummary{},
							},
							{
								ColumnIdentification: columnMetalAMD64OVN,
								Status:               tier1.ExtremeRegression,
								RegressedTests: []crtype.ReportTestSummary{
									{
										ReportTestIdentification: tier1.ReportTestIdentification{
											RowIdentification: tier1.RowIdentification{
												TestName: testName2,
											},
											ColumnIdentification: tier1.ColumnIdentification{
												Variants: columnMetalAMD64OVN.Variants,
											},
										},
										ReportTestStats: testdetails.ReportTestStats{
											ReportStatus: tier1.ExtremeRegression,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedResult: map[JiraComponent][]crtype.ReportTestSummary{
				{Project: "OCPBUGS", Component: "component 1"}: {
					{
						ReportTestIdentification: tier1.ReportTestIdentification{
							RowIdentification: tier1.RowIdentification{
								TestName: testName1,
							},
							ColumnIdentification: tier1.ColumnIdentification{
								Variants: awsAMD64OVNTest.Variants,
							},
						},
						ReportTestStats: testdetails.ReportTestStats{
							ReportStatus: tier1.ExtremeRegression,
						},
					},
				},
				{Project: "OCPBUGS", Component: "Bare Metal Hardware Provisioning"}: {
					{
						ReportTestIdentification: tier1.ReportTestIdentification{
							RowIdentification: tier1.RowIdentification{
								TestName: testName1,
							},
							ColumnIdentification: tier1.ColumnIdentification{
								Variants: columnMetalAMD64OVN.Variants,
							},
						},
						ReportTestStats: testdetails.ReportTestStats{
							ReportStatus: tier1.ExtremeRegression,
						},
					},
					{
						ReportTestIdentification: tier1.ReportTestIdentification{
							RowIdentification: tier1.RowIdentification{
								TestName: testName2,
							},
							ColumnIdentification: tier1.ColumnIdentification{
								Variants: columnMetalAMD64OVN.Variants,
							},
						},
						ReportTestStats: testdetails.ReportTestStats{
							ReportStatus: tier1.ExtremeRegression,
						},
					},
				},
			},
		},
	}
	j := JiraAutomator{
		columnThresholds: map[Variant]int{
			{
				Name:  "Platform",
				Value: "metal",
			}: 1,
		},
		variantToJiraComponents: map[Variant]JiraComponent{
			{
				Name:  "Platform",
				Value: "metal",
			}: {
				Project:   jiratype.ProjectKeyOCPBugs,
				Component: "Bare Metal Hardware Provisioning",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := j.groupRegressedTestsByComponents(tc.report)
			assert.NoError(t, err, "error getting component regressed tests from report")
			fmt.Printf("---- result %+v\n", result)
			assert.Equal(t, tc.expectedResult, result, "expected report %+v, got %+v", tc.expectedResult, result)
		})
	}
}
