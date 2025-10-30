package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &folderResource{}
	_ resource.ResourceWithConfigure = &folderResource{}
)

// NewFolderResource is a helper function to simplify the provider implementation.
func NewFolderResource() resource.Resource {
	return &folderResource{}
}

// folderResource is the resource implementation.
type folderResource struct {
	client *RemoteClient
}

// folderResourceModel maps the resource schema data.
type folderResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Path        types.String `tfsdk:"path"`
	Owner       types.Int64  `tfsdk:"owner"`
	OwnerName   types.String `tfsdk:"owner_name"`
	Group       types.Int64  `tfsdk:"group"`
	GroupName   types.String `tfsdk:"group_name"`
	Permissions types.String `tfsdk:"permissions"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Configure adds the provider configured client to the resource.
func (r *folderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*RemoteClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *hashicups.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *folderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

// Schema defines the schema for the resource.
func (r *folderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Placeholder identifier attribute.",
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"path": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Absolute path to the folder",
			},
			"owner": schema.Int64Attribute{
				Required: false,
				Optional: true,
				Computed: true,
			},
			"permissions": schema.StringAttribute{
				Required: false,
				Optional: true,
				Computed: true,
			},
			"group": schema.Int64Attribute{
				Required: false,
				Optional: true,
				Computed: true,
			},
			"owner_name": schema.StringAttribute{
				Required: false,
				Optional: true,
				Computed: true,
			},
			"group_name": schema.StringAttribute{
				Required: false,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func parseInt(intStr string) int64 {
	v, _ := strconv.ParseInt(intStr, 10, 32)
	return v
}

// Create creates the resource and sets the initial Terraform state.
func (r *folderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan, state folderResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := plan.Path.String()
	state.ID = plan.Path

	err := r.client.CreateDir(path, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating folder",
			"Could not create folder, unexpected error: "+err.Error(),
		)
		return
	}

	if !plan.Owner.IsUnknown() {
		err = r.client.ChownFile(path, plan.Owner.String(), true)
	} else if !plan.OwnerName.IsUnknown() {
		err = r.client.ChownFile(path, plan.OwnerName.String(), true)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating folder user ownership",
			"Could not update, unexpected error: "+err.Error(),
		)
		return
	}

	if !plan.Group.IsUnknown() {
		err = r.client.ChgrpFile(path, plan.Group.String(), true)
	} else if !plan.GroupName.IsUnknown() {
		err = r.client.ChgrpFile(path, plan.GroupName.String(), true)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating folder group ownership",
			"Could not update, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	state.Path = plan.Path
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	group, err := r.client.ReadFileGroup(path, true)
	if err != nil {
        resp.Diagnostics.AddError("Couldn't load dir group id after creation", err.Error())
        return
    }
	owner, err := r.client.ReadFileOwner(path, true)
	if err != nil {
        resp.Diagnostics.AddError("Couldn't load dir owner id after creation", err.Error())
        return
    }
	groupName, err := r.client.ReadFileGroupName(path, true)
	if err != nil {
        resp.Diagnostics.AddError("Couldn't load dir group name after creation", err.Error())
        return
    }
	ownerName, err := r.client.ReadFileOwnerName(path, true)
	if err != nil {
        resp.Diagnostics.AddError("Couldn't load dir owner name after creation", err.Error())
        return
    }
	permissions, err := r.client.ReadFilePermissions(path, true)
	if err != nil {
        resp.Diagnostics.AddError("Couldn't load dir permissions name after creation", err.Error())
        return
    }

	state.Owner = types.Int64Value(parseInt(owner))
	state.Group = types.Int64Value(parseInt(group))
	state.OwnerName = types.StringValue(ownerName)
	state.GroupName = types.StringValue(groupName)
	state.Permissions = types.StringValue(permissions)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *folderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state folderResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := state.ID.ValueString()

	// Get refreshed folder value from HashiCups
	dirExists, err := r.client.dirExists(path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading remote folder",
			"Could not read remote folder ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if !dirExists {
		resp.State.RemoveResource(ctx)
		return
	}

	group, _ := r.client.ReadFileGroup(path, true)
	owner, _ := r.client.ReadFileOwner(path, true)
	groupName, _ := r.client.ReadFileGroupName(path, true)
	ownerName, _ := r.client.ReadFileOwnerName(path, true)
	permissions, _ := r.client.ReadFilePermissions(path, true)

	state.Owner = types.Int64Value(parseInt(owner))
	state.Group = types.Int64Value(parseInt(group))
	state.OwnerName = types.StringValue(ownerName)
	state.GroupName = types.StringValue(groupName)
	state.Permissions = types.StringValue(permissions)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *folderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan, state folderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	path := state.ID.ValueString()

	var err error

	if !plan.Owner.IsUnknown() && plan.Owner != state.Owner {
		err = r.client.ChownFile(path, plan.Owner.String(), true)
	} else if !plan.OwnerName.IsUnknown() && !plan.OwnerName.Equal(state.OwnerName) {
		err = r.client.ChownFile(path, plan.OwnerName.String(), true)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating folder user ownership",
			"Could not update, unexpected error: "+err.Error(),
		)
		return
	}

	if !plan.Group.IsUnknown() && plan.Group != state.Group {
		err = r.client.ChgrpFile(path, plan.Group.String(), true)
	} else if !plan.GroupName.IsUnknown() && !plan.GroupName.Equal(state.GroupName) {
		err = r.client.ChgrpFile(path, plan.GroupName.String(), true)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating folder group ownership",
			"Could not update, unexpected error: "+err.Error(),
		)
		return
	}

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	group, _ := r.client.ReadFileGroup(path, true)
	owner, _ := r.client.ReadFileOwner(path, true)
	groupName, _ := r.client.ReadFileGroupName(path, true)
	ownerName, _ := r.client.ReadFileOwnerName(path, true)
	permissions, _ := r.client.ReadFilePermissions(path, true)

	state.Owner = types.Int64Value(parseInt(owner))
	state.Group = types.Int64Value(parseInt(group))
	state.OwnerName = types.StringValue(ownerName)
	state.GroupName = types.StringValue(groupName)
	state.Permissions = types.StringValue(permissions)

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *folderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state folderResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.DeleteFolder(state.ID.ValueString(), true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting HashiCups Order",
			"Could not delete order, unexpected error: "+err.Error(),
		)
		return
	}
}
