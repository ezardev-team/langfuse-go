package otel

import (
	"testing"

	"github.com/ezardev-team/langfuse-go/model"
)

// TestUsageDetailsMergeOverridesFixed verifies that map keys in
// Usage.UsageDetails are merged into the emitted usage_details and override the
// fixed-field defaults on collision (so a caller can replace the aggregate
// input/output with non-overlapping component counts).
func TestUsageDetailsMergeOverridesFixed(t *testing.T) {
	u := model.Usage{
		Input: 1000, // overridden below
		Total: 1500,
		Unit:  model.ModelUsageUnitTokens,
		UsageDetails: map[string]int{
			"input":                   700, // non-cached portion (overrides 1000)
			"cache_read_input_tokens": 300,
			"output":                  400,
			"output_reasoning":        100,
		},
	}

	got := usageDetails(u)

	if got["input"] != 700 {
		t.Errorf("input = %v; want 700 (map override of fixed 1000)", got["input"])
	}
	if got["cache_read_input_tokens"] != 300 {
		t.Errorf("cache_read_input_tokens = %v; want 300", got["cache_read_input_tokens"])
	}
	if got["output"] != 400 {
		t.Errorf("output = %v; want 400", got["output"])
	}
	if got["output_reasoning"] != 100 {
		t.Errorf("output_reasoning = %v; want 100", got["output_reasoning"])
	}
	if got["total"] != 1500 {
		t.Errorf("total = %v; want 1500 (fixed field preserved)", got["total"])
	}
	if got["unit"] != model.ModelUsageUnitTokens {
		t.Errorf("unit = %v; want TOKENS", got["unit"])
	}
}

// TestCostDetailsMerge verifies precomputed cost lines are merged and that
// hasCost reflects a map-only cost payload.
func TestCostDetailsMerge(t *testing.T) {
	u := model.Usage{
		CostDetails: map[string]float64{
			"input":                   0.0021,
			"cache_read_input_tokens": 0.0001,
			"output":                  0.004,
			"output_reasoning":        0.001,
		},
	}

	if !hasCost(u) {
		t.Fatal("hasCost = false; want true when CostDetails is non-empty")
	}

	got := costDetails(u)
	for k, want := range map[string]float64{
		"input":                   0.0021,
		"cache_read_input_tokens": 0.0001,
		"output":                  0.004,
		"output_reasoning":        0.001,
	} {
		if got[k] != want {
			t.Errorf("cost[%q] = %v; want %v", k, got[k], want)
		}
	}
}

// TestIsZeroUsage covers the explicit (non-comparable) zero check now that
// Usage holds map fields.
func TestIsZeroUsage(t *testing.T) {
	if !isZeroUsage(model.Usage{}) {
		t.Error("empty Usage should be zero")
	}
	if isZeroUsage(model.Usage{UsageDetails: map[string]int{"output_reasoning": 5}}) {
		t.Error("Usage with only UsageDetails should NOT be zero")
	}
	if isZeroUsage(model.Usage{CostDetails: map[string]float64{"output": 0.1}}) {
		t.Error("Usage with only CostDetails should NOT be zero")
	}
	if isZeroUsage(model.Usage{Input: 1}) {
		t.Error("Usage with a fixed field set should NOT be zero")
	}
}
