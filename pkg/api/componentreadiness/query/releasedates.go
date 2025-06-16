package query

import (
	"context"

	"github.com/openshift/sippy/pkg/api"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/requestoptions"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/test"
	"github.com/openshift/sippy/pkg/apis/cache"
	"github.com/openshift/sippy/pkg/bigquery"
	"github.com/openshift/sippy/pkg/util"
)

func GetReleaseDatesFromBigQuery(ctx context.Context, client *bigquery.Client, reqOptions requestoptions.RequestOptions) ([]test.Release, []error) {
	queries := &releaseDateQuerier{client: client, reqOptions: reqOptions}
	return api.GetDataFromCacheOrGenerate[[]test.Release](ctx,
		client.Cache,
		cache.RequestOptions{},
		api.GetPrefixedCacheKey("CRReleaseDates~", reqOptions),
		queries.QueryReleaseDates, []test.Release{})
}

type releaseDateQuerier struct {
	client     *bigquery.Client
	reqOptions requestoptions.RequestOptions
}

func (c *releaseDateQuerier) QueryReleaseDates(ctx context.Context) ([]test.Release, []error) {
	releases, err := api.GetReleasesFromBigQuery(ctx, c.client)
	if err != nil {
		return nil, []error{err}
	}
	crReleases := []test.Release{}
	for _, release := range releases {
		crRelease := test.Release{Release: release.Release}
		if release.GADate != nil {
			prior := util.AdjustReleaseTime(*release.GADate, true, "30", c.reqOptions.CacheOption.CRTimeRoundingFactor)
			crRelease.Start = &prior
			crRelease.End = release.GADate
		}
		crReleases = append(crReleases, crRelease)
	}
	return crReleases, nil
}
