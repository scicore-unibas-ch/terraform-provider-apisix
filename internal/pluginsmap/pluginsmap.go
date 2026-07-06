// Package pluginsmap is the shared codec for APISIX plugin maps: Terraform
// MapAttribute of plugin-name -> JSON-encoded config string on one side,
// map[string]json.RawMessage in the Admin API body on the other.
package pluginsmap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Build converts a plugins map attribute to wire form. Each value is validated
// as JSON so a malformed plugin config fails with an attribute-scoped
// diagnostic instead of being silently dropped. A null/unknown map yields nil
// (field omitted from the request body).
func Build(ctx context.Context, m types.Map, attrPath path.Path) (map[string]json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m.IsNull() || m.IsUnknown() {
		return nil, diags
	}

	plugins := map[string]string{}
	diags.Append(m.ElementsAs(ctx, &plugins, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := make(map[string]json.RawMessage, len(plugins))
	for k, v := range plugins {
		var probe any
		if err := json.Unmarshal([]byte(v), &probe); err != nil {
			diags.AddAttributeError(
				attrPath.AtMapKey(k),
				"Invalid plugin JSON",
				fmt.Sprintf("plugin %q: %v", k, err),
			)
			return nil, diags
		}
		out[k] = json.RawMessage(v)
	}
	return out, diags
}

// Decode converts wire plugin configs to a map attribute value. Each config is
// re-marshaled canonically (Go sorts JSON object keys recursively) so the
// JSON-equivalence plan modifier and ImportStateVerify comparisons stay
// stable; unparsable values are kept verbatim. When the API returned no
// plugins, emptyAsNull selects between a null map (Optional attributes) and an
// empty map value (Required attributes).
func Decode(ctx context.Context, plugins map[string]json.RawMessage, emptyAsNull bool) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics

	strs := make(map[string]string, len(plugins))
	for k, v := range plugins {
		var obj any
		if err := json.Unmarshal(v, &obj); err != nil {
			strs[k] = string(v)
			continue
		}
		canonical, err := json.Marshal(obj)
		if err != nil {
			strs[k] = string(v)
			continue
		}
		strs[k] = string(canonical)
	}

	if len(strs) == 0 && emptyAsNull {
		return types.MapNull(types.StringType), diags
	}
	val, d := types.MapValueFrom(ctx, types.StringType, strs)
	diags.Append(d...)
	return val, diags
}
