// Package timeoutshelper centralizes the small amount of glue needed to apply
// per-resource Timeouts blocks. Each resource exposes the standard
//
//	timeouts {
//	  create = "5m"
//	  read   = "30s"
//	  update = "5m"
//	  delete = "1m"
//	}
//
// configuration; the helper turns the configured value (or a sensible
// fallback) into a context.Context with the appropriate deadline.
package timeoutshelper

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Default is the fallback timeout used when the user hasn't configured one.
// APISIX Admin API operations are sub-second in practice; one minute gives
// plenty of headroom for slow networks while still failing fast on a
// genuinely hung server (vs. Terraform's 20-minute default).
const Default = time.Minute

// Apply returns a derived context with the deadline appropriate for the named
// operation ("create", "read", "update", "delete"). Diagnostics from parsing
// the user-supplied timeout are appended to diags. If diags has an error, the
// caller should return early; the returned cancel func should still be called
// in a defer for cleanup.
func Apply(ctx context.Context, tv timeouts.Value, op string, fallback time.Duration, diags *diag.Diagnostics) (context.Context, context.CancelFunc) {
	var d time.Duration
	var ds diag.Diagnostics
	switch op {
	case "create":
		d, ds = tv.Create(ctx, fallback)
	case "read":
		d, ds = tv.Read(ctx, fallback)
	case "update":
		d, ds = tv.Update(ctx, fallback)
	case "delete":
		d, ds = tv.Delete(ctx, fallback)
	default:
		d = fallback
	}
	diags.Append(ds...)
	if d <= 0 {
		d = fallback
	}
	return context.WithTimeout(ctx, d)
}
