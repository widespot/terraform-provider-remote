package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
	"os"
	"os/user"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &hashicupsProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &hashicupsProvider{
			version: version,
		}
	}
}

// hashicupsProvider is the provider implementation.
type hashicupsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *hashicupsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "remote"
	resp.Version = p.version
}

// hashicupsProviderModel maps provider schema data to a Go type.
type hashicupsProviderModel struct {
	Host             types.String `tfsdk:"host"`
	Username         types.String `tfsdk:"username"`
	Password         types.String `tfsdk:"password"`
	PasswordEnvVar   types.String `tfsdk:"password_env_var"`
	PrivateKey       types.String `tfsdk:"private_key"`
	PrivateKeyPath   types.String `tfsdk:"private_key_path"`
	PrivateKeyEnvVar types.String `tfsdk:"private_key_env_var"`
}

// Schema defines the provider-level schema for configuration data.
func (p *hashicupsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "Remote host to connect. example: `localhost:8022`.",
				Required:    true,
			},
			"username": schema.StringAttribute{
				Description: "SSH user. Default is current user",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "SSH password.",
				Optional:    true,
				Sensitive:   true,
			},
			"password_env_var": schema.StringAttribute{
				Description: "Env var for password.",
				Optional:    true,
				Sensitive:   true,
			},
			"private_key": schema.StringAttribute{
				Description: "SSH private key",
				Optional:    true,
				Sensitive:   true,
			},
			"private_key_path": schema.StringAttribute{
				Description: "Path to SSH private key",
				Optional:    true,
			},
			"private_key_env_var": schema.StringAttribute{
				Description: "Env var with private key",
				Optional:    true,
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *hashicupsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config hashicupsProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var username string
	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	} else {
		// Default value is current user
		currentUser, _ := user.Current()
		username = currentUser.Username
	}

	// Create a new remote client
	clientConfig := ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if !config.Password.IsNull() {
		clientConfig.Auth = append(clientConfig.Auth, ssh.Password(config.Password.ValueString()))
	} else if !config.PasswordEnvVar.IsNull() {
		password := os.Getenv(config.PasswordEnvVar.ValueString())
		if password == "" {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("password_env_var"),
				"Empty password ENV var",
				"",
			)
		}
		clientConfig.Auth = append(clientConfig.Auth, ssh.Password(password))
	}

	if !config.PrivateKey.IsNull() {
		signer, err := ssh.ParsePrivateKey([]byte(config.PrivateKey.ValueString()))
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("private_key"),
				"Private key parsing error",
				fmt.Sprintf("couldn't create a ssh client config from private key: %s", err.Error()),
			)
		}
		clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
	} else if !config.PrivateKeyPath.IsNull() {
		content, err := os.ReadFile(config.PrivateKeyPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("private_key_path"),
				"Private key path reading error",
				fmt.Sprintf("couldn't read private key: %s", err.Error()),
			)
		}
		signer, err := ssh.ParsePrivateKey(content)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("private_key_path"),
				"Private key parsing error",
				fmt.Sprintf("couldn't create a ssh client config from private key: %s", err.Error()),
			)
		}
		clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := NewRemoteClient(config.Host.ValueString(), &clientConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Remote API Client",
			"An unexpected error occurred when creating the HashiCups API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"HashiCups Client Error: "+err.Error(),
		)
		return
	}

	// Make the HashiCups client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *hashicupsProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *hashicupsProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewFolderResource,
		NewFileResource,
	}
}
