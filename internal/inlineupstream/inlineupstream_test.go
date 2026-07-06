package inlineupstream

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDecodeFields_DefaultsOnEmptyObject(t *testing.T) {
	f, diags := DecodeFields(context.Background(), json.RawMessage(`{}`))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}

	for name, got := range map[string]types.String{
		"type":      f.Type,
		"scheme":    f.Scheme,
		"hash_on":   f.HashOn,
		"pass_host": f.PassHost,
	} {
		if got.IsNull() {
			t.Errorf("%s should be backfilled with its schema default, got null", name)
		}
	}
	if f.Type.ValueString() != "roundrobin" || f.Scheme.ValueString() != "http" ||
		f.HashOn.ValueString() != "vars" || f.PassHost.ValueString() != "pass" {
		t.Errorf("defaults wrong: type=%s scheme=%s hash_on=%s pass_host=%s",
			f.Type, f.Scheme, f.HashOn, f.PassHost)
	}

	if !f.Name.IsNull() || !f.Nodes.IsNull() || !f.Timeout.IsNull() ||
		!f.KeepalivePool.IsNull() || !f.TLS.IsNull() || !f.HealthCheck.IsNull() ||
		!f.Labels.IsNull() || !f.Retries.IsNull() {
		t.Errorf("absent fields should decode as null: %+v", f)
	}
}

func TestDecodeFields_LegacyNodesMapIsNull(t *testing.T) {
	raw := json.RawMessage(`{"nodes":{"127.0.0.1:8080":1}}`)
	f, diags := DecodeFields(context.Background(), raw)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !f.Nodes.IsNull() {
		t.Errorf("legacy map-form nodes should decode as null, got %v", f.Nodes)
	}
}

func TestDecodeFields_HealthCheckCanonicalized(t *testing.T) {
	// Keys out of order on the wire must come back sorted.
	raw := json.RawMessage(`{"checks":{"active":{"timeout":1,"http_path":"/ping"}}}`)
	f, diags := DecodeFields(context.Background(), raw)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	want := `{"active":{"http_path":"/ping","timeout":1}}`
	if f.HealthCheck.ValueString() != want {
		t.Errorf("health_check = %s, want %s", f.HealthCheck.ValueString(), want)
	}
}

// fullWire is a representative APISIX upstream GET body exercising every block.
const fullWire = `{
	"name": "backend",
	"desc": "d",
	"type": "chash",
	"scheme": "https",
	"hash_on": "header",
	"key": "x-user",
	"pass_host": "rewrite",
	"upstream_host": "internal.example",
	"retries": 2,
	"retry_timeout": 6,
	"labels": {"env": "test"},
	"nodes": [
		{"host": "10.0.0.1", "port": 8080, "weight": 1, "priority": 0},
		{"host": "10.0.0.2", "port": 8080, "weight": 2, "priority": -1, "metadata": {"az": "a"}}
	],
	"timeout": {"connect": 3, "send": 3, "read": 6},
	"keepalive_pool": {"size": 128, "idle_timeout": 30, "requests": 500},
	"tls": {"client_cert_id": "cert-1", "verify": true},
	"checks": {"active": {"http_path": "/health"}}
}`

func TestRoundTrip_DecodeThenBuild(t *testing.T) {
	ctx := context.Background()
	f, diags := DecodeFields(ctx, json.RawMessage(fullWire))
	if diags.HasError() {
		t.Fatalf("decode diags: %v", diags)
	}

	body, diags := f.Body(ctx, path.Root("upstream"))
	if diags.HasError() {
		t.Fatalf("build diags: %v", diags)
	}
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got, want map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal round-tripped body: %v", err)
	}
	if err := json.Unmarshal(json.RawMessage(fullWire), &want); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip diverged:\n got: %s\nwant: %s", out, fullWire)
	}
}

func TestDecodeInto_ObjectMatchesAttrTypes(t *testing.T) {
	obj, diags := DecodeInto(context.Background(), json.RawMessage(fullWire))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if obj.IsNull() || obj.IsUnknown() {
		t.Fatal("object should be known and non-null")
	}
	attrs := obj.Attributes()
	if got := attrs["type"].(types.String).ValueString(); got != "chash" {
		t.Errorf("type = %q", got)
	}
	nodes := attrs["nodes"].(types.List)
	if len(nodes.Elements()) != 2 {
		t.Errorf("nodes = %v, want 2 elements", nodes)
	}
}

func TestBody_InvalidHealthCheckDiagnosticPath(t *testing.T) {
	f := Fields{HealthCheck: types.StringValue("{not json")}
	_, diags := f.Body(context.Background(), path.Root("upstream"))
	if !diags.HasError() {
		t.Fatal("expected a diagnostic for invalid health_check JSON")
	}
	found := false
	for _, d := range diags.Errors() {
		if strings.Contains(d.Summary(), "Invalid health_check JSON") {
			found = true
		}
	}
	if !found {
		t.Errorf("missing attribute-scoped diagnostic, got: %v", diags)
	}
}

func TestBuildBody_NullObjectIsNil(t *testing.T) {
	body, diags := BuildBody(context.Background(), types.ObjectNull(AttrTypes), path.Root("upstream"))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if body != nil {
		t.Errorf("null object should produce nil body, got %+v", body)
	}
}

func TestTimeoutCodec_RoundTrip(t *testing.T) {
	obj, diags := DecodeTimeout(json.RawMessage(`{"connect":3,"read":6}`))
	if diags.HasError() {
		t.Fatalf("decode diags: %v", diags)
	}
	attrs := obj.Attributes()
	if attrs["connect"].(types.Int64).ValueInt64() != 3 {
		t.Errorf("connect = %v", attrs["connect"])
	}
	if !attrs["send"].(types.Int64).IsNull() {
		t.Errorf("send should be null when absent, got %v", attrs["send"])
	}

	body, diags := BuildTimeout(context.Background(), obj)
	if diags.HasError() {
		t.Fatalf("build diags: %v", diags)
	}
	out, _ := json.Marshal(body)
	if string(out) != `{"connect":3,"read":6}` {
		t.Errorf("wire form = %s", out)
	}
}
