// Package consumerplugins implements the apisix_consumer_plugins resource:
// plugin configuration attached to a consumer that some OTHER system owns.
//
// Why this exists, and why it is not just apisix_consumer:
//
//	apisix_consumer manages the whole object, credentials included. That is
//	wrong when a portal/IdP mints the consumer and its key-auth credential at
//	first login: whichever writer ran last would win, and the user would lose
//	their key. This resource manages ONLY the named plugin keys and leaves
//	every other field — username, group_id, labels, and crucially any
//	credential plugin — exactly as it found it.
//
// Why read-modify-write instead of PATCH:
//
//	The APISIX Admin API does not support PATCH on /consumers/{username}
//	(GET, PUT and DELETE only — unlike routes, services and consumer groups).
//	So each write does GET → merge the managed plugin keys into the existing
//	body → PUT the whole object back. The repo convention "Update calls Put,
//	never PATCH" therefore still holds.
//
//	This is not atomic: if the owning system rewrites the same consumer
//	between our GET and our PUT, that write is lost. The window is
//	milliseconds and the owning system typically writes the consumer only
//	when creating a user, but do not run an apply in a loop against a busy
//	portal. Owning systems can avoid the overlap entirely by keeping their
//	credentials in Credential objects (/consumers/{u}/credentials/{id},
//	APISIX 3.11+) so key rotation never touches the consumer at all.
//
//	The same race applies to this resource against itself: two instances
//	pointing at one consumer are applied in parallel by Terraform and one
//	write is lost. Use a single resource per consumer — the docs and the
//	advanced example say so explicitly.
package consumerplugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/client"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonmap"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/pluginsmap"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/timeoutshelper"
)

const apiKind = "consumers"

// APISIX echoes these into GET bodies; they are not accepted on write.
var syntheticFields = []string{"id", "create_time", "update_time"}

var (
	_ resource.Resource                = (*Resource)(nil)
	_ resource.ResourceWithConfigure   = (*Resource)(nil)
	_ resource.ResourceWithImportState = (*Resource)(nil)
)

// Resource is the apisix_consumer_plugins resource.
type Resource struct {
	client *client.Client
}

// NewResource is the constructor registered with the provider.
func NewResource() resource.Resource { return &Resource{} }

type model struct {
	ConsumerID types.String   `tfsdk:"consumer_id"`
	Plugins    types.Map      `tfsdk:"plugins"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_consumer_plugins"
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = client.FromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages individual plugins on an APISIX Consumer that is created and owned elsewhere. " +
			"Only the plugin keys listed here are touched; all other fields of the consumer, including credential " +
			"plugins such as key-auth, are preserved. The consumer must already exist.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
			"consumer_id": schema.StringAttribute{
				Required:    true,
				Description: "Username of an existing consumer. Changing this forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"plugins": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Map of plugin name to JSON-encoded configuration. These keys are merged into the consumer's " +
					"existing plugins; removing a key here removes it from the consumer on the next apply.",
				PlanModifiers: []planmodifier.Map{
					jsonmap.SuppressEquivalent(),
				},
			},
		},
	}
}

// fetch returns the consumer object with synthetic fields stripped, ready to
// be modified and PUT back.
func (r *Resource) fetch(ctx context.Context, id string) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	apiResp, err := r.client.Get(ctx, apiKind, id)
	if err != nil {
		return nil, nil, err
	}

	body := map[string]json.RawMessage{}
	if err := json.Unmarshal(apiResp.Value, &body); err != nil {
		return nil, nil, fmt.Errorf("decode consumer response: %w", err)
	}
	for _, f := range syntheticFields {
		delete(body, f)
	}

	existing := map[string]json.RawMessage{}
	if raw, ok := body["plugins"]; ok {
		if err := json.Unmarshal(raw, &existing); err != nil {
			return nil, nil, fmt.Errorf("decode consumer plugins: %w", err)
		}
	}
	return body, existing, nil
}

// write merges managed into the consumer's plugins, drops any key in remove
// that we are no longer managing, and PUTs the result.
func (r *Resource) write(ctx context.Context, id string, managed map[string]json.RawMessage, remove []string) error {
	body, plugins, err := r.fetch(ctx, id)
	if err != nil {
		return err
	}

	for _, k := range remove {
		delete(plugins, k)
	}
	for k, v := range managed {
		plugins[k] = v
	}

	if len(plugins) == 0 {
		delete(body, "plugins")
	} else {
		encoded, err := json.Marshal(plugins)
		if err != nil {
			return fmt.Errorf("encode consumer plugins: %w", err)
		}
		body["plugins"] = encoded
	}

	_, err = r.client.Put(ctx, apiKind, id, body)
	return err
}

// managedKeys returns the plugin names currently recorded in state.
func managedKeys(m types.Map) []string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	keys := make([]string, 0, len(m.Elements()))
	for k := range m.Elements() {
		keys = append(keys, k)
	}
	return keys
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := timeoutshelper.Apply(ctx, plan.Timeouts, "create", timeoutshelper.Default, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	managed, diags := pluginsmap.Build(ctx, plan.Plugins, path.Root("plugins"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.write(ctx, plan.ConsumerID.ValueString(), managed, nil)
	if errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError(
			"Consumer does not exist",
			fmt.Sprintf("Consumer %q was not found. This resource attaches plugins to a consumer created "+
				"elsewhere; create the consumer first (or let the owning system create it).",
				plan.ConsumerID.ValueString()),
		)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to attach consumer plugins", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := timeoutshelper.Apply(ctx, state.Timeouts, "read", timeoutshelper.Default, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	_, existing, err := r.fetch(ctx, state.ConsumerID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read consumer", err.Error())
		return
	}

	// Reflect only the keys we manage. Plugins the owning system or an
	// operator added are none of this resource's business.
	mine := map[string]json.RawMessage{}
	for _, k := range managedKeys(state.Plugins) {
		if v, ok := existing[k]; ok {
			mine[k] = v
		}
	}
	if len(mine) == 0 {
		// Every managed plugin is gone: the resource no longer exists.
		resp.State.RemoveResource(ctx)
		return
	}

	pVal, diags := pluginsmap.Decode(ctx, mine, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Plugins = pVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := timeoutshelper.Apply(ctx, plan.Timeouts, "update", timeoutshelper.Default, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	managed, diags := pluginsmap.Build(ctx, plan.Plugins, path.Root("plugins"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keys dropped from the config must be removed from the consumer, or they
	// would linger forever with their last-applied value.
	var stale []string
	for _, k := range managedKeys(state.Plugins) {
		if _, kept := managed[k]; !kept {
			stale = append(stale, k)
		}
	}

	if err := r.write(ctx, plan.ConsumerID.ValueString(), managed, stale); err != nil {
		resp.Diagnostics.AddError("Failed to update consumer plugins", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := timeoutshelper.Apply(ctx, state.Timeouts, "delete", timeoutshelper.Default, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove our keys and leave the consumer itself alone. A missing consumer
	// means there is nothing left to detach.
	err := r.write(ctx, state.ConsumerID.ValueString(), nil, managedKeys(state.Plugins))
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Failed to detach consumer plugins", err.Error())
	}
}

// ImportState takes "<consumer>/<plugin>[,<plugin>...]", e.g.
// "alice/ai-rate-limiting".
//
// The plugin names are part of the import ID on purpose. Importing by consumer
// alone would pull in every plugin on the object — key-auth included — and the
// first apply would then hand credential ownership to Terraform, which is the
// one thing this resource exists to prevent. Consumer names cannot contain
// "/", so the separator is unambiguous.
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	consumerID, pluginList, found := strings.Cut(req.ID, "/")
	if !found || consumerID == "" || pluginList == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected \"<consumer>/<plugin>[,<plugin>...]\" (for example "+
				"\"alice/ai-rate-limiting\"), got %q. The plugins to manage must be named "+
				"explicitly so that importing cannot take ownership of credential plugins.",
				req.ID),
		)
		return
	}

	_, existing, err := r.fetch(ctx, consumerID)
	if errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Consumer does not exist", fmt.Sprintf("Consumer %q was not found.", consumerID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to import consumer plugins", err.Error())
		return
	}

	mine := map[string]json.RawMessage{}
	for _, name := range strings.Split(pluginList, ",") {
		name = strings.TrimSpace(name)
		v, ok := existing[name]
		if !ok {
			available := make([]string, 0, len(existing))
			for k := range existing {
				available = append(available, k)
			}
			sort.Strings(available)
			resp.Diagnostics.AddError(
				"Plugin not found on consumer",
				fmt.Sprintf("Consumer %q has no plugin %q. Plugins currently on it: %s.",
					consumerID, name, strings.Join(available, ", ")),
			)
			return
		}
		mine[name] = v
	}

	pVal, diags := pluginsmap.Decode(ctx, mine, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("consumer_id"), consumerID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("plugins"), pVal)...)
}
