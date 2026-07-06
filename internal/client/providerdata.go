package client

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// FromProviderData extracts the *Client the provider stored in ResourceData.
// It returns nil when data is nil (Configure not called yet — the framework
// calls resource Configure before provider Configure during validation) and
// appends a diagnostic when data has an unexpected type.
func FromProviderData(data any, diags *diag.Diagnostics) *Client {
	if data == nil {
		return nil
	}
	c, ok := data.(*Client)
	if !ok {
		diags.AddError(
			"Unexpected provider data",
			fmt.Sprintf("expected *client.Client, got %T", data),
		)
		return nil
	}
	return c
}
