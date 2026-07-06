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
