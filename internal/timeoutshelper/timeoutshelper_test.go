package timeoutshelper

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func deadlineIn(t *testing.T, ctx context.Context) time.Duration {
	t.Helper()
	dl, ok := ctx.Deadline()
	if !ok {
		t.Fatal("context has no deadline")
	}
	return time.Until(dl)
}

func TestApply_UnconfiguredUsesFallback(t *testing.T) {
	tv := timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{"create": types.StringType})}
	var diags diag.Diagnostics

	ctx, cancel := Apply(context.Background(), tv, "create", time.Minute, &diags)
	defer cancel()
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if d := deadlineIn(t, ctx); d > time.Minute || d < 50*time.Second {
		t.Errorf("deadline %v not near the 1m fallback", d)
	}
}

func TestApply_ConfiguredValueWins(t *testing.T) {
	obj, d := types.ObjectValue(
		map[string]attr.Type{"read": types.StringType},
		map[string]attr.Value{"read": types.StringValue("2h")},
	)
	if d.HasError() {
		t.Fatalf("object: %v", d)
	}
	tv := timeouts.Value{Object: obj}
	var diags diag.Diagnostics

	ctx, cancel := Apply(context.Background(), tv, "read", time.Minute, &diags)
	defer cancel()
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if d := deadlineIn(t, ctx); d < 90*time.Minute {
		t.Errorf("deadline %v should reflect the configured 2h timeout", d)
	}
}

func TestApply_UnknownOpUsesFallback(t *testing.T) {
	tv := timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{})}
	var diags diag.Diagnostics

	ctx, cancel := Apply(context.Background(), tv, "bogus", 30*time.Second, &diags)
	defer cancel()
	if d := deadlineIn(t, ctx); d > 30*time.Second || d < 20*time.Second {
		t.Errorf("deadline %v not near the 30s fallback", d)
	}
}
