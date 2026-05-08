// Package jsonstring provides a String plan modifier that treats
// JSON-equivalent strings as equal. Useful for fields like apisix_route.vars
// where APISIX returns the JSON with reordered keys / different whitespace.
package jsonstring

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// SuppressEquivalent returns a String plan modifier that suppresses plan
// changes when the planned value is JSON-equivalent to the prior state.
func SuppressEquivalent() planmodifier.String {
	return suppress{}
}

type suppress struct{}

func (suppress) Description(_ context.Context) string {
	return "Treat JSON-equivalent strings as equal so server-normalized JSON does not produce drift."
}

func (s suppress) MarkdownDescription(ctx context.Context) string { return s.Description(ctx) }

func (suppress) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if jsonEqual(req.PlanValue.ValueString(), req.StateValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
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
