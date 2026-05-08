package jsonmap

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mapVal(t *testing.T, m map[string]string) types.Map {
	t.Helper()
	v, diags := types.MapValueFrom(context.Background(), types.StringType, m)
	if diags.HasError() {
		t.Fatalf("MapValueFrom failed: %v", diags)
	}
	return v
}

func runModifier(t *testing.T, state, plan types.Map) types.Map {
	t.Helper()
	ctx := context.Background()
	req := planmodifier.MapRequest{StateValue: state, PlanValue: plan}
	resp := &planmodifier.MapResponse{PlanValue: plan}
	SuppressEquivalent().PlanModifyMap(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	return resp.PlanValue
}

func TestSuppressEquivalent_JSONEquivalentValues(t *testing.T) {
	state := mapVal(t, map[string]string{"plugin": `{"a":1,"b":2}`})
	plan := mapVal(t, map[string]string{"plugin": `{"b":2,"a":1}`})

	got := runModifier(t, state, plan)
	if !got.Equal(state) {
		t.Errorf("plan should have been replaced with state for JSON-equivalent values; got %v want %v", got, state)
	}
}

func TestSuppressEquivalent_DifferentValuesPreserved(t *testing.T) {
	state := mapVal(t, map[string]string{"plugin": `{"a":1}`})
	plan := mapVal(t, map[string]string{"plugin": `{"a":2}`})

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be preserved when values differ; got %v", got)
	}
}

func TestSuppressEquivalent_DifferentKeysPreserved(t *testing.T) {
	state := mapVal(t, map[string]string{"a": `{"x":1}`})
	plan := mapVal(t, map[string]string{"b": `{"x":1}`})

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be preserved when keys differ; got %v", got)
	}
}

func TestSuppressEquivalent_NullStateNoOp(t *testing.T) {
	state := types.MapNull(types.StringType)
	plan := mapVal(t, map[string]string{"plugin": `{"a":1}`})

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be unchanged when state is null; got %v", got)
	}
}

func TestSuppressEquivalent_NullPlanNoOp(t *testing.T) {
	state := mapVal(t, map[string]string{"plugin": `{"a":1}`})
	plan := types.MapNull(types.StringType)

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be unchanged when plan is null; got %v", got)
	}
}

func TestSuppressEquivalent_InvalidJSONPreserved(t *testing.T) {
	state := mapVal(t, map[string]string{"plugin": `{"a":1}`})
	plan := mapVal(t, map[string]string{"plugin": `not-json`})

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be preserved when one side is invalid JSON; got %v", got)
	}
}

func TestSuppressEquivalent_PartialEquivalence(t *testing.T) {
	// One key is JSON-equivalent, another is not. The non-equivalent one
	// must keep the planned value; the equivalent one re-uses the state value.
	state := mapVal(t, map[string]string{
		"equiv":   `{"a":1,"b":2}`,
		"changed": `{"x":1}`,
	})
	plan := mapVal(t, map[string]string{
		"equiv":   `{"b":2,"a":1}`,
		"changed": `{"x":2}`,
	})

	got := runModifier(t, state, plan)
	want := mapVal(t, map[string]string{
		"equiv":   `{"a":1,"b":2}`,
		"changed": `{"x":2}`,
	})
	if !got.Equal(want) {
		t.Errorf("partial equivalence wrong; got %v want %v", got, want)
	}
}
