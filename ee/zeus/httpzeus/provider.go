package httpzeus

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/http/client"
	"github.com/SigNoz/signoz/pkg/zeus"
)

type Provider struct {
	settings   factory.ScopedProviderSettings
	config     zeus.Config
	httpClient *client.Client
}

func NewProviderFactory() factory.ProviderFactory[zeus.Zeus, zeus.Config] {
	return factory.NewProviderFactory(factory.MustNewName("http"), func(ctx context.Context, providerSettings factory.ProviderSettings, config zeus.Config) (zeus.Zeus, error) {
		return New(ctx, providerSettings, config)
	})
}

func New(ctx context.Context, providerSettings factory.ProviderSettings, config zeus.Config) (zeus.Zeus, error) {
	settings := factory.NewScopedProviderSettings(providerSettings, "github.com/SigNoz/signoz/ee/zeus/httpzeus")

	httpClient, err := client.New(
		settings.Logger(),
		providerSettings.TracerProvider,
		providerSettings.MeterProvider,
		client.WithRequestResponseLog(true),
		client.WithRetryCount(3),
	)
	if err != nil {
		return nil, err
	}

	return &Provider{
		settings:   settings,
		config:     config,
		httpClient: httpClient,
	}, nil
}

func (provider *Provider) GetLicense(ctx context.Context, key string) ([]byte, error) {
	// Bypass license check for local development
	return []byte(`{
		"id": "0196f794-ff30-7bee-a5f4-ef5ad315715e",
		"key": "local-enterprise-key",
		"category": "ENTERPRISE",
		"status": "ACTIVE",
		"plan": {
			"name": "ENTERPRISE"
		},
		"valid_from": 1700000000,
		"valid_until": 4863266400,
		"state": "active"
	}`), nil
}

func (provider *Provider) GetCheckoutURL(ctx context.Context, key string, body []byte) ([]byte, error) {
	return []byte(`"http://localhost:8080/checkout-dummy"`), nil
}

func (provider *Provider) GetPortalURL(ctx context.Context, key string, body []byte) ([]byte, error) {
	return []byte(`"http://localhost:8080/portal-dummy"`), nil
}

func (provider *Provider) GetDeployment(ctx context.Context, key string) ([]byte, error) {
	return []byte(`{"status": "active"}`), nil
}

func (provider *Provider) PutMeters(ctx context.Context, key string, data []byte) error {
	return nil
}

func (provider *Provider) PutProfile(ctx context.Context, key string, body []byte) error {
	return nil
}

func (provider *Provider) PutHost(ctx context.Context, key string, body []byte) error {
	return nil
}

func (provider *Provider) do(ctx context.Context, url *url.URL, method string, key string, requestBody []byte) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, url.String(), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Signoz-Cloud-Api-Key", key)
	request.Header.Set("Content-Type", "application/json")

	response, err := provider.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode/100 == 2 {
		return body, nil
	}

	return nil, provider.errFromStatusCode(response.StatusCode)
}

// This can be taken down to the client package
func (provider *Provider) errFromStatusCode(statusCode int) error {
	switch statusCode {
	case http.StatusBadRequest:
		return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "bad request")
	case http.StatusUnauthorized:
		return errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "unauthenticated")
	case http.StatusForbidden:
		return errors.Newf(errors.TypeForbidden, errors.CodeForbidden, "forbidden")
	case http.StatusNotFound:
		return errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "not found")
	}

	return errors.Newf(errors.TypeInternal, errors.CodeInternal, "internal")
}
