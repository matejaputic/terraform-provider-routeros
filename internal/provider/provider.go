// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matejaputic/terraform-provider-routeros/internal/client"
	"os"
	"strconv"
	"time"
)

type RouterOSProvider struct{ version string }
type configModel struct {
	HostURL  types.String `tfsdk:"hosturl"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	CA       types.String `tfsdk:"ca_certificate"`
	Insecure types.Bool   `tfsdk:"insecure"`
	Timeout  types.Int64  `tfsdk:"rest_timeout"`
}

func (p *RouterOSProvider) Metadata(_ context.Context, _ provider.MetadataRequest, r *provider.MetadataResponse) {
	r.TypeName = "routeros"
	r.Version = p.version
}
func (p *RouterOSProvider) Schema(_ context.Context, _ provider.SchemaRequest, r *provider.SchemaResponse) {
	r.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"hosturl": schema.StringAttribute{Optional: true}, "username": schema.StringAttribute{Optional: true}, "password": schema.StringAttribute{Optional: true, Sensitive: true}, "ca_certificate": schema.StringAttribute{Optional: true, Description: "Path to PEM CA certificate file."}, "insecure": schema.BoolAttribute{Optional: true}, "rest_timeout": schema.Int64Attribute{Optional: true, Description: "HTTP timeout in seconds, minimum 5; default 59."},
	}}
}
func env(keys ...string) string {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
	}
	return ""
}
func fallback(v types.String, keys ...string) string {
	if v.IsNull() {
		return env(keys...)
	}
	return v.ValueString()
}
func (p *RouterOSProvider) Configure(ctx context.Context, q provider.ConfigureRequest, r *provider.ConfigureResponse) {
	var m configModel
	r.Diagnostics.Append(q.Config.Get(ctx, &m)...)
	if r.Diagnostics.HasError() {
		return
	}
	if m.HostURL.IsUnknown() || m.Username.IsUnknown() || m.Password.IsUnknown() || m.CA.IsUnknown() || m.Insecure.IsUnknown() || m.Timeout.IsUnknown() {
		r.Diagnostics.AddError("Unknown provider configuration", "Provider configuration must be known before configuring RouterOS.")
		return
	}
	c := client.Config{HostURL: fallback(m.HostURL, "ROS_HOSTURL", "MIKROTIK_HOST"), Username: fallback(m.Username, "ROS_USERNAME", "MIKROTIK_USER"), Password: fallback(m.Password, "ROS_PASSWORD", "MIKROTIK_PASSWORD"), Timeout: 59 * time.Second, Insecure: m.Insecure.ValueBool()}
	if m.Insecure.IsNull() {
		if s := env("ROS_INSECURE", "MIKROTIK_INSECURE"); s != "" {
			b, e := strconv.ParseBool(s)
			if e != nil {
				r.Diagnostics.AddError("Invalid insecure environment value", "Expected a boolean.")
				return
			}
			c.Insecure = b
		}
	}
	if !m.Timeout.IsNull() {
		if m.Timeout.ValueInt64() < 5 || m.Timeout.ValueInt64() > 86400 {
			r.Diagnostics.AddError("Invalid rest_timeout", "Expected 5 to 86400 seconds.")
			return
		}
		c.Timeout = time.Duration(m.Timeout.ValueInt64()) * time.Second
	}
	if path := fallback(m.CA, "ROS_CA_CERTIFICATE", "MIKROTIK_CA_CERTIFICATE"); path != "" {
		b, e := os.ReadFile(path)
		if e != nil {
			r.Diagnostics.AddError("Invalid CA certificate", "Unable to read CA certificate file.")
			return
		}
		c.CA = string(b)
	}
	if c.HostURL == "" || c.Username == "" {
		r.Diagnostics.AddError("Missing provider configuration", "Set hosturl and username, or their ROS_/MIKROTIK_ environment variables.")
		return
	}
	cl, e := client.New(c)
	if e != nil {
		r.Diagnostics.AddError("Invalid RouterOS configuration", e.Error())
		return
	}
	// Resolve version once before sharing this instance with resources. Aliases
	// have separate clients; no global RouterOS version or stored context.
	var system map[string]any
	if e := cl.Request(ctx, "GET", "/system/resource", "", nil, nil, &system); e != nil {
		cl.Close()
		r.Diagnostics.AddError("RouterOS connection failed", e.Error())
		return
	}
	version, ok := system["version"].(string)
	if !ok || version == "" {
		cl.Close()
		r.Diagnostics.AddError("Invalid RouterOS response", "System resource response did not include a version.")
		return
	}
	reviewed, err := client.CheckRuntimeVersion(version)
	if err != nil {
		cl.Close()
		r.Diagnostics.AddError("Unsupported RouterOS version", err.Error())
		return
	}
	if !reviewed {
		r.Diagnostics.AddWarning("Unverified RouterOS lane", "Only RouterOS 7.24.5 x86_64/base has live acceptance evidence. This version is experimental; package and architecture compatibility are not inferred.")
	}
	var packages []map[string]any
	if err := cl.Request(ctx, "GET", "/system/package", "", nil, nil, &packages); err != nil {
		cl.Close()
		r.Diagnostics.AddError("RouterOS package discovery failed", err.Error())
		return
	}
	installed, err := client.ValidatePackages(version, packages)
	if err != nil {
		cl.Close()
		r.Diagnostics.AddError("Unsupported RouterOS packages", err.Error())
		return
	}
	cl.Version = version
	cl.Packages = installed
	r.ResourceData = cl
	r.DataSourceData = cl
}
func (p *RouterOSProvider) Resources(context.Context) []func() resource.Resource {
	return append(append(append(append([]func() resource.Resource{NewIPAddressResource}, collectionConstructors()...), singletonConstructors()...), extendedCollectionConstructors()...), additionalSettingsConstructors()...)
}
func (p *RouterOSProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &RouterOSProvider{version: version} }
}
