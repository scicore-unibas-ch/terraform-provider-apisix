// Package jsonmap provides plan modifiers for Map attributes whose values are
// JSON-encoded strings (e.g. APISIX plugin configuration maps).
//
// APISIX returns plugin objects with server-injected defaults and re-ordered keys.
// Without suppression, every refresh produces a spurious plan diff. The plan
// modifier here parses each map value as JSON and, if logically equal to the
// prior state value, re-uses the prior state value so no plan diff is produced.
package jsonmap

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SuppressEquivalent returns a Map plan modifier that treats JSON-equivalent
// string values as equal.
func SuppressEquivalent() planmodifier.Map {
	return suppress{}
}

type suppress struct{}

func (suppress) Description(_ context.Context) string {
	return "Treat JSON-equivalent string values as equal so server-normalized JSON does not produce drift."
}

func (s suppress) MarkdownDescription(ctx context.Context) string { return s.Description(ctx) }

func (suppress) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	stateMap := map[string]string{}
	planMap := map[string]string{}
	resp.Diagnostics.Append(req.StateValue.ElementsAs(ctx, &stateMap, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &planMap, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !sameKeys(planMap, stateMap) {
		return
	}

	merged := make(map[string]string, len(planMap))
	for k, planVal := range planMap {
		stateVal := stateMap[k]
		if jsonEqual(planVal, stateVal) {
			merged[k] = stateVal
		} else {
			merged[k] = planVal
		}
	}

	newVal, d := types.MapValueFrom(ctx, types.StringType, merged)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.PlanValue = newVal
}

func sameKeys(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func jsonEqual(a, b string) bool {
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}
