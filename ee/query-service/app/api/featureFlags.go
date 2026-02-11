package api

import (
	"net/http"

	"github.com/SigNoz/signoz/ee/query-service/constants"
	"github.com/SigNoz/signoz/pkg/flagger"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/featuretypes"
	"github.com/SigNoz/signoz/pkg/types/licensetypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

func (ah *APIHandler) getFeatureFlags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(w, err)
		return
	}

	orgID := valuer.MustNewUUID(claims.OrgID)

	// Cloud-only: all features are always enabled.
	featureSet := make([]*licensetypes.Feature, len(licensetypes.AllFeatures))
	copy(featureSet, licensetypes.AllFeatures)

	// Flagger-based config-driven flags still evaluated at runtime.
	evalCtx := featuretypes.NewFlaggerEvaluationContext(orgID)
	useSpanMetrics := ah.Signoz.Flagger.BooleanOrEmpty(ctx, flagger.FeatureUseSpanMetrics, evalCtx)

	featureSet = append(featureSet, &licensetypes.Feature{
		Name:       valuer.NewString(flagger.FeatureUseSpanMetrics.String()),
		Active:     useSpanMetrics,
		Usage:      0,
		UsageLimit: -1,
		Route:      "",
	})

	if constants.IsDotMetricsEnabled {
		for idx, feature := range featureSet {
			if feature.Name == licensetypes.DotMetricsEnabled {
				featureSet[idx].Active = true
			}
		}
	}

	ah.Respond(w, featureSet)
}
