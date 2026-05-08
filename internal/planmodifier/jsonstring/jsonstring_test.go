package jsonstring

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runModifier(t *testing.T, state, plan types.String) types.String {
	t.Helper()
	ctx := context.Background()
	req := planmodifier.StringRequest{StateValue: state, PlanValue: plan}
	resp := &planmodifier.StringResponse{PlanValue: plan}
	SuppressEquivalent().PlanModifyString(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	return resp.PlanValue
}

func TestSuppressEquivalent_KeyOrderIgnored(t *testing.T) {
	state := types.StringValue(`{"a":1,"b":2}`)
	plan := types.StringValue(`{"b":2,"a":1}`)

	got := runModifier(t, state, plan)
	if !got.Equal(state) {
		t.Errorf("plan should have been replaced with state for JSON-equivalent strings; got %v want %v", got, state)
	}
}

func TestSuppressEquivalent_NumericTypeEquivalence(t *testing.T) {
	// JSON spec treats 1 and 1.0 as numerically equal — Go's
	// reflect.DeepEqual on json.Unmarshal output keeps them both as
	// float64, so they should compare equal.
	state := types.StringValue(`{"x":1}`)
	plan := types.StringValue(`{"x":1.0}`)

	got := runModifier(t, state, plan)
	if !got.Equal(state) {
		t.Errorf("plan should have been replaced with state for numeric-equivalent JSON; got %v want %v", got, state)
	}
}

func TestSuppressEquivalent_DifferentValuesPreserved(t *testing.T) {
	state := types.StringValue(`{"a":1}`)
	plan := types.StringValue(`{"a":2}`)

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be preserved when values differ; got %v", got)
	}
}

func TestSuppressEquivalent_NullStateNoOp(t *testing.T) {
	state := types.StringNull()
	plan := types.StringValue(`{"a":1}`)

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be unchanged when state is null; got %v", got)
	}
}

func TestSuppressEquivalent_NullPlanNoOp(t *testing.T) {
	state := types.StringValue(`{"a":1}`)
	plan := types.StringNull()

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be unchanged when plan is null; got %v", got)
	}
}

func TestSuppressEquivalent_InvalidJSONPreserved(t *testing.T) {
	state := types.StringValue(`{"a":1}`)
	plan := types.StringValue(`not-json`)

	got := runModifier(t, state, plan)
	if !got.Equal(plan) {
		t.Errorf("plan should be preserved when one side is invalid JSON; got %v", got)
	}
}

func TestSuppressEquivalent_NestedEquivalence(t *testing.T) {
	state := types.StringValue(`[["a","==",1],["b","in",[1,2,3]]]`)
	plan := types.StringValue(`[["a","==",1],["b","in",[1,2,3]]]`)

	got := runModifier(t, state, plan)
	if !got.Equal(state) {
		t.Errorf("identical strings should remain stable; got %v", got)
	}
}
