package httplicensing

import (
	"context"
	"encoding/json"

	"github.com/SigNoz/signoz/ee/licensing/licensingstore/sqllicensingstore"
	"github.com/SigNoz/signoz/pkg/analytics"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/licensing"
	"github.com/SigNoz/signoz/pkg/modules/organization"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/licensetypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/SigNoz/signoz/pkg/zeus"
	"github.com/tidwall/gjson"
)

type provider struct {
	store     licensetypes.Store
	zeus      zeus.Zeus
	settings  factory.ScopedProviderSettings
	orgGetter organization.Getter
	stopChan  chan struct{}
}

func NewProviderFactory(store sqlstore.SQLStore, zeus zeus.Zeus, orgGetter organization.Getter, analytics analytics.Analytics) factory.ProviderFactory[licensing.Licensing, licensing.Config] {
	return factory.NewProviderFactory(factory.MustNewName("http"), func(ctx context.Context, providerSettings factory.ProviderSettings, config licensing.Config) (licensing.Licensing, error) {
		return New(ctx, providerSettings, config, store, zeus, orgGetter, analytics)
	})
}

func New(ctx context.Context, ps factory.ProviderSettings, config licensing.Config, sqlstore sqlstore.SQLStore, zeus zeus.Zeus, orgGetter organization.Getter, _ analytics.Analytics) (licensing.Licensing, error) {
	settings := factory.NewScopedProviderSettings(ps, "github.com/SigNoz/signoz/ee/licensing/httplicensing")
	licensestore := sqllicensingstore.New(sqlstore)
	return &provider{
		store:     licensestore,
		zeus:      zeus,
		settings:  settings,
		orgGetter: orgGetter,
		stopChan:  make(chan struct{}),
	}, nil
}

// Start is a no-op. No periodic license validation in cloud-only mode.
func (provider *provider) Start(ctx context.Context) error {
	<-provider.stopChan
	return nil
}

func (provider *provider) Stop(ctx context.Context) error {
	provider.settings.Logger().DebugContext(ctx, "licensing provider stopped")
	close(provider.stopChan)
	return nil
}

// Validate is a no-op. No license validation in cloud-only mode.
func (provider *provider) Validate(ctx context.Context) error {
	return nil
}

// Activate stores a license key for billing purposes.
func (provider *provider) Activate(ctx context.Context, organizationID valuer.UUID, key string) error {
	data, err := provider.zeus.GetLicense(ctx, key)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, errors.CodeInternal, "unable to fetch license data with upstream server")
	}

	license, err := licensetypes.NewLicense(data, organizationID)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, errors.CodeInternal, "failed to create license entity")
	}

	storableLicense := licensetypes.NewStorableLicenseFromLicense(license)
	return provider.store.Create(ctx, storableLicense)
}

// GetActive returns the stored license (for billing key), or a synthetic stub if none exists.
func (provider *provider) GetActive(ctx context.Context, organizationID valuer.UUID) (*licensetypes.License, error) {
	storableLicenses, err := provider.store.GetAll(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	activeLicense, err := licensetypes.GetActiveLicenseFromStorableLicenses(storableLicenses, organizationID)
	if err != nil {
		if errors.Ast(err, errors.TypeNotFound) {
			return licensetypes.NewSyntheticCloudLicense(organizationID), nil
		}
		return nil, err
	}

	return activeLicense, nil
}

// Refresh is a no-op. No license validation in cloud-only mode.
func (provider *provider) Refresh(ctx context.Context, organizationID valuer.UUID) error {
	return nil
}

// Checkout creates a billing checkout session using the stored license key.
func (provider *provider) Checkout(ctx context.Context, organizationID valuer.UUID, postableSubscription *licensetypes.PostableSubscription) (*licensetypes.GettableSubscription, error) {
	activeLicense, err := provider.GetActive(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(postableSubscription)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, errors.CodeInvalidInput, "failed to marshal checkout payload")
	}

	response, err := provider.zeus.GetCheckoutURL(ctx, activeLicense.Key, body)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, errors.CodeInternal, "failed to generate checkout session")
	}

	return &licensetypes.GettableSubscription{RedirectURL: gjson.GetBytes(response, "url").String()}, nil
}

// Portal creates a billing portal session using the stored license key.
func (provider *provider) Portal(ctx context.Context, organizationID valuer.UUID, postableSubscription *licensetypes.PostableSubscription) (*licensetypes.GettableSubscription, error) {
	activeLicense, err := provider.GetActive(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(postableSubscription)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, errors.CodeInvalidInput, "failed to marshal portal payload")
	}

	response, err := provider.zeus.GetPortalURL(ctx, activeLicense.Key, body)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, errors.CodeInternal, "failed to generate portal session")
	}

	return &licensetypes.GettableSubscription{RedirectURL: gjson.GetBytes(response, "url").String()}, nil
}

// GetFeatureFlags returns all features as active. No license lookup needed.
func (provider *provider) GetFeatureFlags(ctx context.Context, organizationID valuer.UUID) ([]*licensetypes.Feature, error) {
	return licensetypes.AllFeatures, nil
}

// Collect returns empty stats. No license-based analytics in cloud-only mode.
func (provider *provider) Collect(ctx context.Context, orgID valuer.UUID) (map[string]any, error) {
	return map[string]any{}, nil
}
