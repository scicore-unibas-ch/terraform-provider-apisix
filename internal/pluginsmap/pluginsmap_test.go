package pluginsmap

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mapVal(t *testing.T, m map[string]string) types.Map {
	t.Helper()
	v, diags := types.MapValueFrom(context.Background(), types.StringType, m)
	if diags.HasError() {
		t.Fatalf("MapValueFrom: %v", diags)
	}
	return v
}

func TestBuild_NullMapIsNil(t *testing.T) {
	out, diags := Build(context.Background(), types.MapNull(types.StringType), path.Root("plugins"))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if out != nil {
		t.Errorf("null map should produce nil, got %v", out)
	}
}

func TestBuild_ValidPluginsPassThrough(t *testing.T) {
	in := mapVal(t, map[string]string{"limit-count": `{"count":10,"time_window":60}`})
	out, diags := Build(context.Background(), in, path.Root("plugins"))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if string(out["limit-count"]) != `{"count":10,"time_window":60}` {
		t.Errorf("out = %v", out)
	}
}

func TestBuild_InvalidJSONFailsWithScopedDiagnostic(t *testing.T) {
	in := mapVal(t, map[string]string{"bad-plugin": `{oops`})
	_, diags := Build(context.Background(), in, path.Root("plugins"))
	if !diags.HasError() {
		t.Fatal("expected diagnostic for invalid plugin JSON")
	}
	found := false
	for _, d := range diags.Errors() {
		if strings.Contains(d.Detail(), "bad-plugin") {
			found = true
		}
	}
	if !found {
		t.Errorf("diagnostic should name the offending plugin: %v", diags)
	}
}

func TestDecode_CanonicalizesValues(t *testing.T) {
	in := map[string]json.RawMessage{"p": json.RawMessage(`{"b":2,"a":1}`)}
	out, diags := Decode(context.Background(), in, false)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	var got map[string]string
	_ = out.ElementsAs(context.Background(), &got, false)
	if got["p"] != `{"a":1,"b":2}` {
		t.Errorf("value = %q, want canonical key order", got["p"])
	}
}

func TestDecode_EmptyHandling(t *testing.T) {
	asNull, diags := Decode(context.Background(), nil, true)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !asNull.IsNull() {
		t.Errorf("emptyAsNull=true should yield a null map, got %v", asNull)
	}

	asEmpty, diags := Decode(context.Background(), nil, false)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if asEmpty.IsNull() || len(asEmpty.Elements()) != 0 {
		t.Errorf("emptyAsNull=false should yield an empty map value, got %v", asEmpty)
	}
}

func TestDecode_UnparsableValueKeptVerbatim(t *testing.T) {
	in := map[string]json.RawMessage{"p": json.RawMessage(`{broken`)}
	out, diags := Decode(context.Background(), in, false)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	var got map[string]string
	_ = out.ElementsAs(context.Background(), &got, false)
	if got["p"] != `{broken` {
		t.Errorf("value = %q, want verbatim passthrough", got["p"])
	}
}
