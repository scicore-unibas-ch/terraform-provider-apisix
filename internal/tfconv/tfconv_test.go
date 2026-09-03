package tfconv

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCanonicalJSON(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		wantNull bool
		want     string
	}{
		{"empty is null", "", true, ""},
		{"json null is null", "null", true, ""},
		{"keys sorted recursively", `{"b":{"y":1,"x":2},"a":3}`, false, `{"a":3,"b":{"x":2,"y":1}}`},
		{"array preserved", `[{"b":1,"a":2}]`, false, `[{"a":2,"b":1}]`},
		{"invalid kept verbatim", `{oops`, false, `{oops`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanonicalJSON(json.RawMessage(tc.raw))
			if tc.wantNull {
				if !got.IsNull() {
					t.Errorf("want null, got %v", got)
				}
				return
			}
			if got.ValueString() != tc.want {
				t.Errorf("got %q, want %q", got.ValueString(), tc.want)
			}
		})
	}
}

func TestPtrConversions(t *testing.T) {
	if StringPtr(types.StringNull()) != nil || StringPtr(types.StringUnknown()) != nil {
		t.Error("null/unknown string must convert to nil")
	}
	if v := StringPtr(types.StringValue("")); v == nil || *v != "" {
		t.Error("explicit empty string must convert to non-nil pointer")
	}
	if Int64Ptr(types.Int64Null()) != nil {
		t.Error("null int64 must convert to nil")
	}
	if v := Int64Ptr(types.Int64Value(0)); v == nil || *v != 0 {
		t.Error("explicit zero must convert to non-nil pointer")
	}
	if BoolPtr(types.BoolNull()) != nil {
		t.Error("null bool must convert to nil")
	}
	if v := BoolPtr(types.BoolValue(false)); v == nil || *v {
		t.Error("explicit false must convert to non-nil pointer")
	}
}

func TestNullableAndDefaults(t *testing.T) {
	if !NullableString("").IsNull() {
		t.Error(`NullableString("") should be null`)
	}
	if NullableString("x").ValueString() != "x" {
		t.Error("NullableString should keep non-empty values")
	}
	if StringOrDefault("", "d").ValueString() != "d" {
		t.Error("StringOrDefault should substitute the default")
	}
	if !OptInt64(nil).IsNull() {
		t.Error("OptInt64(nil) should be null")
	}
	if Int64OrDefault(nil, 42).ValueInt64() != 42 {
		t.Error("Int64OrDefault should substitute the default")
	}
	empty := ""
	if !OptString(&empty).IsNull() {
		t.Error("OptString of empty string should be null")
	}
}

// The Admin API cannot distinguish an absent field from one set to "", so the
// decode path has to take its cue from the prior state. Getting this wrong is
// not a cosmetic problem: it produces a resource that reports "will be updated
// in-place" on every plan for ever.
func TestNullableStringPreserving(t *testing.T) {
	// The practitioner wrote desc = "". Keep it, or config "" and state null
	// disagree on every subsequent plan.
	if got := NullableStringPreserving("", types.StringValue("")); got.IsNull() {
		t.Error(`an explicit "" in prior state must survive an empty API value`)
	}

	// Nothing was ever set. Absent stays absent.
	if got := NullableStringPreserving("", types.StringNull()); !got.IsNull() {
		t.Error("an absent field with null prior state must stay null")
	}

	// Import: there is no prior state, so the zero value (null) applies.
	var zero types.String
	if got := NullableStringPreserving("", zero); !got.IsNull() {
		t.Error("on import (zero prior) an empty API value must be null")
	}

	// A real value always wins, whatever the prior representation was.
	for _, prior := range []types.String{types.StringNull(), types.StringValue(""), types.StringValue("old")} {
		if got := NullableStringPreserving("x", prior); got.ValueString() != "x" {
			t.Errorf("a non-empty API value must win, prior=%v", prior)
		}
	}

	// Drift: the value was removed at the API. Prior "" is still "" — both mean
	// "no description" and neither should churn the plan.
	if got := NullableStringPreserving("", types.StringUnknown()); !got.IsNull() {
		t.Error("unknown prior must not be echoed back into state")
	}
}
