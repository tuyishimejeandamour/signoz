package licensetypes

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/uptrace/bun"
)

type StorableLicense struct {
	bun.BaseModel `bun:"table:license"`

	types.Identifiable
	types.TimeAuditable
	Key             string         `bun:"key,type:text,notnull,unique"`
	Data            map[string]any `bun:"data,type:text"`
	LastValidatedAt time.Time      `bun:"last_validated_at,notnull"`
	OrgID           valuer.UUID    `bun:"org_id,type:text,notnull" json:"orgId"`
}

// this data excludes ID and Key
type License struct {
	ID              valuer.UUID
	Key             string
	Data            map[string]interface{}
	PlanName        valuer.String
	Features        []*Feature
	Status          valuer.String
	State           string
	FreeUntil       time.Time
	ValidFrom       int64
	ValidUntil      int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastValidatedAt time.Time
	OrganizationID  valuer.UUID
}

type GettableLicense map[string]any

type PostableLicense struct {
	Key string `json:"key"`
}

func NewStorableLicense(ID valuer.UUID, key string, data map[string]any, createdAt, updatedAt, lastValidatedAt time.Time, organizationID valuer.UUID) *StorableLicense {
	return &StorableLicense{
		Identifiable: types.Identifiable{
			ID: ID,
		},
		TimeAuditable: types.TimeAuditable{
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Key:             key,
		Data:            data,
		LastValidatedAt: lastValidatedAt,
		OrgID:           organizationID,
	}
}

func NewStorableLicenseFromLicense(license *License) *StorableLicense {
	return &StorableLicense{
		Identifiable: types.Identifiable{
			ID: license.ID,
		},
		TimeAuditable: types.TimeAuditable{
			CreatedAt: license.CreatedAt,
			UpdatedAt: license.UpdatedAt,
		},
		Key:             license.Key,
		Data:            license.Data,
		LastValidatedAt: license.LastValidatedAt,
		OrgID:           license.OrganizationID,
	}
}

func GetActiveLicenseFromStorableLicenses(storableLicenses []*StorableLicense, organizationID valuer.UUID) (*License, error) {
	var activeLicense *License
	for _, storableLicense := range storableLicenses {
		license, err := NewLicenseFromStorableLicense(storableLicense)
		if err != nil {
			return nil, err
		}

		if activeLicense == nil &&
			(license.ValidFrom != 0) &&
			(license.ValidUntil == -1 || license.ValidUntil > time.Now().Unix()) {
			activeLicense = license
		}
		if activeLicense != nil &&
			license.ValidFrom > activeLicense.ValidFrom &&
			(license.ValidUntil == -1 || license.ValidUntil > time.Now().Unix()) {
			activeLicense = license
		}
	}

	if activeLicense == nil {
		return nil, errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "no active license found for the organization %s", organizationID.StringValue())
	}

	return activeLicense, nil
}

func extractKeyFromMapStringInterface[T any](data map[string]interface{}, key string) (T, error) {
	var zeroValue T
	if val, ok := data[key]; ok {
		if value, ok := val.(T); ok {
			return value, nil
		}
		return zeroValue, errors.NewInvalidInputf(errors.CodeInvalidInput, "%s key is not a valid %s", key, reflect.TypeOf(zeroValue))
	}
	return zeroValue, errors.NewInvalidInputf(errors.CodeInvalidInput, "%s key is missing", key)
}

func NewLicense(data []byte, organizationID valuer.UUID) (*License, error) {
	licenseData := map[string]any{}
	err := json.Unmarshal(data, &licenseData)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, errors.CodeInternal, "failed to unmarshal license data")
	}

	var features []*Feature

	// extract id from data
	licenseIDStr, err := extractKeyFromMapStringInterface[string](licenseData, "id")
	if err != nil {
		return nil, err
	}
	licenseID, err := valuer.NewUUID(licenseIDStr)
	if err != nil {
		return nil, err
	}
	delete(licenseData, "id")

	// extract key from data
	licenseKey, err := extractKeyFromMapStringInterface[string](licenseData, "key")
	if err != nil {
		return nil, err
	}
	delete(licenseData, "key")

	// extract status from data
	statusStr, err := extractKeyFromMapStringInterface[string](licenseData, "status")
	if err != nil {
		return nil, err
	}
	status := valuer.NewString(statusStr)

	planMap, err := extractKeyFromMapStringInterface[map[string]any](licenseData, "plan")
	if err != nil {
		return nil, err
	}

	planNameStr, err := extractKeyFromMapStringInterface[string](planMap, "name")
	if err != nil {
		return nil, err
	}
	planName := valuer.NewString(planNameStr)

	state, err := extractKeyFromMapStringInterface[string](licenseData, "state")
	if err != nil {
		state = ""
	}

	freeUntilStr, err := extractKeyFromMapStringInterface[string](licenseData, "free_until")
	if err != nil {
		freeUntilStr = ""
	}

	freeUntil, err := time.Parse(time.RFC3339, freeUntilStr)
	if err != nil {
		freeUntil = time.Time{}
	}

	// Cloud-only: all features always enabled regardless of plan.
	features = append(features, AllFeatures...)
	licenseData["features"] = features

	_validFrom, err := extractKeyFromMapStringInterface[float64](licenseData, "valid_from")
	if err != nil {
		_validFrom = 0
	}
	validFrom := int64(_validFrom)

	_validUntil, err := extractKeyFromMapStringInterface[float64](licenseData, "valid_until")
	if err != nil {
		_validUntil = 0
	}
	validUntil := int64(_validUntil)

	return &License{
		ID:              licenseID,
		Key:             licenseKey,
		Data:            licenseData,
		PlanName:        planName,
		Features:        features,
		ValidFrom:       validFrom,
		ValidUntil:      validUntil,
		Status:          status,
		State:           state,
		FreeUntil:       freeUntil,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LastValidatedAt: time.Now(),
		OrganizationID:  organizationID,
	}, nil

}

func NewLicenseFromStorableLicense(storableLicense *StorableLicense) (*License, error) {
	var features []*Feature
	// extract status from data
	statusStr, err := extractKeyFromMapStringInterface[string](storableLicense.Data, "status")
	if err != nil {
		return nil, err
	}
	status := valuer.NewString(statusStr)

	planMap, err := extractKeyFromMapStringInterface[map[string]any](storableLicense.Data, "plan")
	if err != nil {
		return nil, err
	}

	planNameStr, err := extractKeyFromMapStringInterface[string](planMap, "name")
	if err != nil {
		return nil, err
	}
	planName := valuer.NewString(planNameStr)

	// Cloud-only: all features always enabled regardless of plan.
	features = append(features, AllFeatures...)
	storableLicense.Data["features"] = features

	_validFrom, err := extractKeyFromMapStringInterface[float64](storableLicense.Data, "valid_from")
	if err != nil {
		_validFrom = 0
	}
	validFrom := int64(_validFrom)

	_validUntil, err := extractKeyFromMapStringInterface[float64](storableLicense.Data, "valid_until")
	if err != nil {
		_validUntil = 0
	}
	validUntil := int64(_validUntil)

	state, err := extractKeyFromMapStringInterface[string](storableLicense.Data, "state")
	if err != nil {
		state = ""
	}

	freeUntilStr, err := extractKeyFromMapStringInterface[string](storableLicense.Data, "free_until")
	if err != nil {
		freeUntilStr = ""
	}

	freeUntil, err := time.Parse(time.RFC3339, freeUntilStr)
	if err != nil {
		freeUntil = time.Time{}
	}

	return &License{
		ID:              storableLicense.ID,
		Key:             storableLicense.Key,
		Data:            storableLicense.Data,
		PlanName:        planName,
		Features:        features,
		ValidFrom:       validFrom,
		ValidUntil:      validUntil,
		Status:          status,
		State:           state,
		FreeUntil:       freeUntil,
		CreatedAt:       storableLicense.CreatedAt,
		UpdatedAt:       storableLicense.UpdatedAt,
		LastValidatedAt: storableLicense.LastValidatedAt,
		OrganizationID:  storableLicense.OrgID,
	}, nil

}

func NewStatsFromLicense(license *License) map[string]any {
	return map[string]any{
		"license.id":              license.ID.StringValue(),
		"license.plan.name":       license.PlanName.StringValue(),
		"license.state.name":      license.State,
		"license.free_until.time": license.FreeUntil.UTC(),
	}
}

func (license *License) UpdateFeatures(features []*Feature) {
	license.Features = features
}

func (license *License) Update(data []byte) error {
	updatedLicense, err := NewLicense(data, license.OrganizationID)
	if err != nil {
		return err
	}

	currentTime := time.Now()
	license.Data = updatedLicense.Data
	license.Features = updatedLicense.Features
	license.ID = updatedLicense.ID
	license.Key = updatedLicense.Key
	license.PlanName = updatedLicense.PlanName
	license.Status = updatedLicense.Status
	license.ValidFrom = updatedLicense.ValidFrom
	license.ValidUntil = updatedLicense.ValidUntil
	license.UpdatedAt = currentTime
	license.LastValidatedAt = currentTime

	return nil
}

// NewSyntheticCloudLicense returns a stub license when no real license is stored.
// Key is empty so callers that need a real key for external services can detect and degrade gracefully.
func NewSyntheticCloudLicense(organizationID valuer.UUID) *License {
	now := time.Now()
	createdAt := now.UTC().Format(time.RFC3339)
	data := map[string]any{
		"key":        "",
		"status":     "VALID",
		"state":      "ACTIVE",
		"platform":   "CLOUD",
		"created_at": createdAt,
		"updated_at": createdAt,
		"plan": map[string]any{
			"name":        "cloud",
			"is_active":   true,
			"description": "",
			"created_at":  createdAt,
			"updated_at":  createdAt,
		},
		"plan_id":    "",
		"free_until": "",
		"valid_from":  float64(0),
		"valid_until": float64(-1),
		"event_queue": map[string]any{
			"event":        "",
			"status":       "",
			"scheduled_at": "",
			"created_at":   createdAt,
			"updated_at":   createdAt,
		},
		"features": AllFeatures,
	}
	return &License{
		ID:              valuer.UUID{},
		Key:             "",
		Data:            data,
		PlanName:        valuer.NewString("cloud"),
		Features:        AllFeatures,
		Status:          valuer.NewString("VALID"),
		State:           "ACTIVE",
		ValidFrom:       0,
		ValidUntil:      -1,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastValidatedAt: now,
		OrganizationID:  organizationID,
	}
}

func NewGettableLicense(data map[string]any, key string) *GettableLicense {
	gettableLicense := make(GettableLicense)
	for k, v := range data {
		gettableLicense[k] = v
	}
	gettableLicense["key"] = key
	return &gettableLicense
}

func (p *PostableLicense) UnmarshalJSON(data []byte) error {
	var postableLicense struct {
		Key string `json:"key"`
	}

	err := json.Unmarshal(data, &postableLicense)
	if err != nil {
		return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "failed to unmarshal payload")
	}

	if postableLicense.Key == "" {
		return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "license key cannot be empty")
	}

	p.Key = postableLicense.Key
	return nil
}

type Store interface {
	Create(context.Context, *StorableLicense) error
	Get(context.Context, valuer.UUID, valuer.UUID) (*StorableLicense, error)
	GetAll(context.Context, valuer.UUID) ([]*StorableLicense, error)
	Update(context.Context, valuer.UUID, *StorableLicense) error
}
