package tier1

const (
	// FailedFixedRegression indicates someone has claimed the bug is fix, but we see failures past the resolution time
	FailedFixedRegression Status = -1000
	// ExtremeRegression shows regression with >15% pass rate change
	ExtremeRegression Status = -500
	// SignificantRegression shows significant regression
	SignificantRegression Status = -400
	// ExtremeTriagedRegression shows an ExtremeRegression that clears when Triaged incidents are factored in
	ExtremeTriagedRegression Status = -300
	// SignificantTriagedRegression shows a SignificantRegression that clears when Triaged incidents are factored in
	SignificantTriagedRegression Status = -200
	// FixedRegression indicates someone has claimed the bug is now fixed, but has not yet rolled off the sample window
	FixedRegression Status = -150
	// MissingSample indicates sample data missing
	MissingSample Status = -100
	// NotSignificant indicates no significant difference
	NotSignificant Status = 0
	// MissingBasis indicates basis data missing
	MissingBasis Status = 100
	// MissingBasisAndSample indicates basis and sample data missing
	MissingBasisAndSample Status = 200
	// SignificantImprovement indicates improved sample rate
	SignificantImprovement Status = 300
)

const (
	PassRate    Comparison = "pass_rate"
	FisherExact Comparison = "fisher_exact"
)
