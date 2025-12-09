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

	// FindCachedGenerationBatch searches for multiple GENERATION observations that match the provided cache keys.
	// Returns a map of cacheKey -> ObservationView for found entries. Keys not found are omitted from the result.
	FindCachedGenerationBatch(ctx context.Context, cacheKeys []string, options *GenerationCacheOptions) (map[string]*model.ObservationView, error)
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

	return l.findCachedGeneration(ctx, cacheKey, options)
}

// FindCachedGenerationBatch searches for multiple GENERATION observations that match the provided cache keys.
// Returns a map of cacheKey -> ObservationView for found entries. Keys not found are omitted from the result.
func (l *Langfuse) FindCachedGenerationBatch(ctx context.Context, cacheKeys []string, options *GenerationCacheOptions) (map[string]*model.ObservationView, error) {
	if len(cacheKeys) == 0 {
		return make(map[string]*model.ObservationView), nil
	}

	result := make(map[string]*model.ObservationView)

	for _, cacheKey := range cacheKeys {
		if cacheKey == "" {
			return nil, fmt.Errorf("cache key is required")
		}

		obs, err := l.findCachedGeneration(ctx, cacheKey, options)
		if err != nil {
			return nil, err
		}

		if obs != nil {
			result[cacheKey] = obs
		}
	}

	return result, nil
}

func (l *Langfuse) findCachedGeneration(ctx context.Context, cacheKey string, options *GenerationCacheOptions) (*model.ObservationView, error) {
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
	orderBy := "startTime"
	req := api.ObservationsRequest{
		Page:    &page,
		Limit:   &limit,
		Filter:  string(filterString),
		OrderBy: &orderBy,
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

	for i := range res.Data {
		obs := &res.Data[i]
		foundCacheKey, ok := extractCacheKeyFromMetadata(obs.Metadata)
		if !ok || foundCacheKey != cacheKey {
			continue
		}

		return obs, nil
	}

	return nil, nil
}

func extractCacheKeyFromMetadata(metadata any) (string, bool) {
	switch m := metadata.(type) {
	case map[string]interface{}:
		if val, ok := m[cacheMetadataKey]; ok {
			cacheKey, ok := val.(string)
			return cacheKey, ok
		}
	case model.M:
		if val, ok := m[cacheMetadataKey]; ok {
			cacheKey, ok := val.(string)
			return cacheKey, ok
		}
	}

	return "", false
}
