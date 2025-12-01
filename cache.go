package langfuse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ezardev-team/langfuse-go/internal/pkg/api"
	"github.com/ezardev-team/langfuse-go/model"
)

const cacheMetadataKey = "cache_key"

// GenerationCache defines the cache lookup helper on top of the public API client.
type GenerationCache interface {
	// FindCachedGeneration searches for a GENERATION observation that matches the provided cache key
	// (and optionally a specific observation name). It returns nil when no cached result exists.
	FindCachedGeneration(ctx context.Context, cacheKey string, options *GenerationCacheOptions) (*model.ObservationView, error)
}

type GenerationCacheOptions struct {
	// Name optionally scopes the lookup to a specific observation/function name.
	Name string
}

type observationFilter struct {
	Type     string      `json:"type"`
	Column   string      `json:"column"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value,omitempty"`
	Key      string      `json:"key,omitempty"`
}

var _ GenerationCache = (*Langfuse)(nil)

// FindCachedGeneration implements GenerationCache and returns the first matching GENERATION observation output for a cache key.
func (l *Langfuse) FindCachedGeneration(ctx context.Context, cacheKey string, options *GenerationCacheOptions) (*model.ObservationView, error) {
	if cacheKey == "" {
		return nil, fmt.Errorf("cache key is required")
	}

	filters := []observationFilter{
		{
			Type:     "string",
			Column:   "type",
			Operator: "=",
			Value:    model.ObservationTypeGeneration,
		},
		{
			Type:     "stringObject",
			Column:   "metadata",
			Key:      cacheMetadataKey,
			Operator: "=",
			Value:    cacheKey,
		},
	}

	if options != nil && options.Name != "" {
		filters = append(filters, observationFilter{
			Type:     "string",
			Column:   "name",
			Operator: "=",
			Value:    options.Name,
		})
	}

	filterString, err := json.Marshal(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to encode observation filters: %w", err)
	}

	limit := 1
	page := 1
	req := api.ObservationsRequest{
		Page:   &page,
		Limit:  &limit,
		Filter: string(filterString),
	}

	res := api.ObservationsResponse{}
	if err := l.client.Observations(ctx, &req, &res); err != nil {
		return nil, err
	}

	if !res.IsSuccess() {
		if res.RawBody != nil {
			return nil, fmt.Errorf("observations request failed with status code: %d body=%s", res.Code, *res.RawBody)
		}
		return nil, fmt.Errorf("observations request failed with status code: %d", res.Code)
	}

	if len(res.Data) == 0 {
		return nil, nil
	}

	return &res.Data[0], nil
}
