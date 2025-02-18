package sippyserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			risk := makeNewTestRisk("test1", 2, c.tests)
			assert.Equal(t, c.expected.Failures, risk.Failures)
			assert.Equal(t, c.expected.Flakes, risk.Flakes)
			assert.Equal(t, c.expected.AnyMissing, risk.AnyMissing)
		})
	}
}
