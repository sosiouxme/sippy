package models

import (
	"time"

	"gorm.io/gorm"
)

type ReleaseTag struct {
	Model

	// ReleaseTag contains the release version, e.g. 4.8.0-0.nightly-2021-10-28-013428.
	ReleaseTag string `json:"release_tag" gorm:"column:release_tag"`

	// Release contains the release X.Y version, e.g. 4.8
	Release string `json:"release" gorm:"column:release"`

	// Stream contains the payload stream, e.g. nightly or ci.
	Stream string `json:"stream" gorm:"column:stream"`

	// Architecture contains the arch for a release, e.g. amd64
	Architecture string `json:"architecture" gorm:"column:architecture"`

	// Phase contains the overall status of a payload: e.g. Ready, Accepted,
	// Rejected. We do not store Ready payloads in bigquery, as we only want
	// the release after it's "fully baked."
	Phase string `json:"phase" gorm:"column:phase"`

	// ReleaseTime contains the timestamp of the release (the suffix of the tag, -YYYY-MM-DD-HHMMSS).
	ReleaseTime time.Time `json:"release_time" gorm:"column:release_time"`

	// PreviousReleaseTag contains the previously accepted build, on which any
	// changelog is based from.
	PreviousReleaseTag string `json:"previous_release_tag" gorm:"column:previous_release_tag"`

	// KubernetesVersion contains the kube version for this payload.
	KubernetesVersion string `json:"kubernetes_version" gorm:"column:kubernetes_version"`

	// CurrentOSVersion contains the current machine OS version.
	CurrentOSVersion string `json:"current_os_version" gorm:"current_os_version"`

	// PreviousOSVersion, if any, indicates this release included a machine OS
	// upgrade and this field contains the prior version.
	PreviousOSVersion string `json:"previous_os_version" gorm:"previous_os_version"`

	// CurrentOSURL is a link to the release page for this machine OS version.
	CurrentOSURL string `json:"current_os_url" gorm:"current_os_url"`

	// PreviousOSURL is a link to the release page for the previous machine OS version.
	PreviousOSURL string `json:"previous_os_url" gorm:"previous_os_url"`

	// OSDiffURL is a link to the release page diffing the two OS versions.
	OSDiffURL string `json:"os_diff_url" gorm:"os_diff_url"`

	// ReleasePullRequest contains a list of all the PR's in a release.
	PullRequests []ReleasePullRequest `json:"-" gorm:"many2many:release_tag_pull_requests;"`

	Repositories []ReleaseRepository `json:"-" gorm:"foreignKey:release_tag_id"`

	JobRuns []ReleaseJobRun `json:"-" gorm:"foreignKey:release_tag_id"`
}

// ReleasePullRequest represents a pull request that was included for the first time
// in a release payload.
type ReleasePullRequest struct {
	Model

	// URL is a link to the pull request.
	URL string `json:"url" gorm:"index:pr_url_name,unique;column:url"`

	// PullRequestID contains the ID of the GitHub pull request.
	PullRequestID string `json:"pull_request_id" gorm:"column:pull_request_id"`

	// Name contains the names as the repository is known in the release payload.
	Name string `json:"name" gorm:"index:pr_url_name,unique;column:name"`

	// Description is the PR description.
	Description string `json:"description" gorm:"column:description"`

	// BugURL links to the bug, if any.
	BugURL string `json:"bug_url" gorm:"column:bug_url"`
}

type ReleaseRepository struct {
	Model

	// Name of the repository, as known by the release payload.
	Name string `json:"name" gorm:"column:name"`

	// ReleaseTag this specific repository ref references.
	ReleaseTag ReleaseTag `gorm:"foreignKey:release_tag_id"`

	// ReleaseTagID foreign key.
	ReleaseTagID string `json:"release_tag" gorm:"column:release_tag_id"`

	// Head is the SHA of the git repo.
	Head string `json:"repository_head" gorm:"column:repository_head"`

	// DiffURL is a link to the git diff.
	DiffURL string `json:"url" gorm:"column:diff_url"`
}

type ReleaseJobRun struct {
	Model

	ReleaseTag     ReleaseTag `json:"release_tag" gorm:"foreignKey:release_tag_id"`
	ReleaseTagID   string     `gorm:"column:release_tag_id"`
	Name           string     `json:"name" gorm:"column:name"`
	JobName        string     `json:"job_name" gorm:"column:job_name"`
	Kind           string     `json:"kind" gorm:"column:kind"`
	State          string     `json:"state" gorm:"column:state"`
	TransitionTime time.Time  `json:"transition_time"`
	Retries        int        `json:"retries"`
	URL            string     `json:"url" gorm:"column:url"`
	UpgradesFrom   string     `json:"upgrades_from" gorm:"column:upgrades_from"`
	UpgradesTo     string     `json:"upgrades_to" gorm:"column:upgrades_to"`
	Upgrade        bool       `json:"upgrade" gorm:"column:upgrade"`
}

// GetLastAcceptedByArchitectureAndStream returns the last accepted payload for each architecture/stream combo.
func GetLastAcceptedByArchitectureAndStream(db *gorm.DB, release string) ([]ReleaseTag, error) {
	results := make([]ReleaseTag, 0)

	result := db.Raw(`SELECT
						DISTINCT ON
							(architecture, stream)
							*
						FROM
							release_tags
						WHERE
							release = ?
						AND
							phase = 'Accepted'
						ORDER BY
							architecture, stream, release_time desc`, release).Scan(&results)

	if result.Error != nil {
		return nil, result.Error
	}

	return results, nil
}

// GetLastPayloadStatus returns the most recent payload status for an architecture/stream combination,
// as well as the count of how many of the last payloads had that status (e.g., when this returns
// Rejected, 5 -- it means the last 5 payloads were rejected.
func GetLastPayloadStatus(db *gorm.DB, architecture, stream, release string) (string, int, error) {
	count := struct {
		Phase string `gorm:"column:phase"`
		Count int    `gorm:"column:count"`
	}{}

	result := db.Raw(`
		WITH releases AS
			(
				SELECT
					ROW_NUMBER() OVER(ORDER BY release_time desc) AS id,
					phase
				FROM
					release_tags
				WHERE
					architecture = ? AND stream = ? AND release = ?
			),
		changes AS
			(
				SELECT
					*,
					CASE WHEN LAG(phase) OVER(ORDER BY id) = phase THEN 0 ELSE 1 END AS change
				FROM
					releases
			),
		groups AS
			(
				SELECT
					*,
					SUM(change) OVER(ORDER BY id) AS group FROM changes
			)
		SELECT
			phase, COUNT(phase)
		FROM
			groups
		WHERE
			groups.group = 1
		GROUP BY
			phase`, architecture, stream, release).Scan(&count)

	return count.Phase, count.Count, result.Error
}