package query

import (
	"context"

	"github.com/openshift/sippy/pkg/api"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/requestoptions"
	"github.com/openshift/sippy/pkg/apis/api/componentreport/tier1"
	"github.com/openshift/sippy/pkg/apis/cache"
	"github.com/openshift/sippy/pkg/bigquery"
	"github.com/openshift/sippy/pkg/util"
)

func GetReleaseDatesFromBigQuery(ctx context.Context, client *bigquery.Client, reqOptions requestoptions.RequestOptions) ([]tier1.Release, []error) {
	queries := &releaseDateQuerier{client: client, reqOptions: reqOptions}
	return api.GetDataFromCacheOrGenerate[[]tier1.Release](ctx,
		client.Cache,
		cache.RequestOptions{},
		api.GetPrefixedCacheKey("CRReleaseDates~", reqOptions),
		queries.QueryReleaseDates, []tier1.Release{})
}

type releaseDateQuerier struct {
	client     *bigquery.Client
	reqOptions requestoptions.RequestOptions
}

func (c *releaseDateQuerier) QueryReleaseDates(ctx context.Context) ([]tier1.Release, []error) {
	releases, err := api.GetReleasesFromBigQuery(ctx, c.client)
	if err != nil {
		return nil, []error{err}
	}
	crReleases := []tier1.Release{}
	for _, release := range releases {
		crRelease := tier1.Release{Release: release.Release}
		if release.GADate != nil {
			prior := util.AdjustReleaseTime(*release.GADate, true, "30", c.reqOptions.CacheOption.CRTimeRoundingFactor)
			crRelease.Start = &prior
			crRelease.End = release.GADate
		}
		crReleases = append(crReleases, crRelease)
	}
	return crReleases, nil
}
