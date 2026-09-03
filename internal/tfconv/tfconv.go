// Package tfconv holds the small conversions between wire-side Go values and
// Terraform framework types that every resource decode/build path needs.
package tfconv

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NullableString maps the wire convention "absent field decodes as empty
// string" to a null Terraform value.
//
// Prefer NullableStringPreserving in a decode path that has the prior state:
// this function alone cannot tell an absent field from one the practitioner
// explicitly set to "", and collapsing the latter to null causes a permanent
// diff. See NullableStringPreserving.
func NullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// NullableStringPreserving is NullableString, except that it keeps an empty
// string the practitioner actually wrote.
//
// Why this exists. The Admin API does not distinguish "field absent" from
// "field set to an empty string" — both decode as "". The write path
// (StringPtr) treats "" as a real value and sends it; the read path collapses
// "" to null. Those two disagree, and Terraform compares CONFIG against STATE,
// not what went over the wire, so for an Optional attribute:
//
//	config     desc = ""      -> plan ""      -> state "" (set from the plan)
//	next plan  refresh Read   -> state null   -> "" != null
//
// which reports the resource as "will be updated in-place" on every plan, for
// ever, with no attribute rendered because the values are equal once
// normalised. Passing the prior state lets the decode keep the representation
// the configuration chose: "" stays "", genuinely absent stays null.
//
// prior is the value already in state (zero value, i.e. null, on import).
func NullableStringPreserving(s string, prior types.String) types.String {
	if s == "" && !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
		return prior
	}
	return NullableString(s)
}

// StringOrDefault substitutes def when the API omitted the field. Used for
// Optional+Computed attributes whose schema Default must match.
func StringOrDefault(s, def string) types.String {
	if s == "" {
		return types.StringValue(def)
	}
	return types.StringValue(s)
}

// OptString maps a nil-or-empty wire pointer to null.
func OptString(p *string) types.String {
	if p == nil || *p == "" {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

// OptInt64 maps a nil wire pointer to null.
func OptInt64(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

// Int64OrDefault substitutes def when the API omitted the field. Used for
// Optional+Computed attributes whose schema Default must match.
func Int64OrDefault(p *int64, def int64) types.Int64 {
	if p == nil {
		return types.Int64Value(def)
	}
	return types.Int64Value(*p)
}

// StringPtr converts a Terraform value to a wire pointer: null/unknown becomes
// nil (field omitted from the request body).
func StringPtr(s types.String) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	v := s.ValueString()
	return &v
}

// Int64Ptr converts a Terraform value to a wire pointer: null/unknown becomes
// nil (field omitted from the request body).
func Int64Ptr(i types.Int64) *int64 {
	if i.IsNull() || i.IsUnknown() {
		return nil
	}
	v := i.ValueInt64()
	return &v
}

// BoolPtr converts a Terraform value to a wire pointer: null/unknown becomes
// nil (field omitted from the request body).
func BoolPtr(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

// CanonicalJSON stores a raw JSON API field as a string attribute value.
// Absent / JSON-null fields become null; anything else is re-marshaled through
// map ordering so key order is canonical (Go sorts JSON object keys
// recursively), keeping ImportStateVerify text comparisons stable. Values that
// fail to parse are stored verbatim.
func CanonicalJSON(raw json.RawMessage) types.String {
	if len(raw) == 0 || string(raw) == "null" {
		return types.StringNull()
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return types.StringValue(string(raw))
	}
	canon, err := json.Marshal(v)
	if err != nil {
		return types.StringValue(string(raw))
	}
	return types.StringValue(string(canon))
}
