package provider

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &EvmProvider{}

type EvmClient interface {
	bind.ContractBackend
	bind.DeployBackend
	ChainID(context.Context) (*big.Int, error)
}

// EvmProvider defines the provider implementation.
type EvmProvider struct {
	client EvmClient
}

// EvmProviderModel describes the provider data model.
type EvmProviderModel struct {
	NodeUrl types.String `tfsdk:"node_url"`
}

func (p *EvmProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "evm"
}

func (p *EvmProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"node_url": schema.StringAttribute{
				MarkdownDescription: "EVM node URL",
				Optional:            true,
			},
		},
	}
}

func (p *EvmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config EvmProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if p.client == nil {
		if config.NodeUrl.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("node_url"),
				"Missing Node URL",
				"URL to EVM RPC Node required",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if p.client == nil {
		var client EvmClient
		client, err := ethclient.DialContext(ctx, config.NodeUrl.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create EVM RPC Client",
				"Unexpected error when creating EVM RPC Client.\n\n"+
					"EVM Client Error: "+err.Error(),
			)
			return
		}
		p.client = client
	}

	resp.DataSourceData = p.client
	resp.ResourceData = p.client
}

func (p *EvmProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRandomPkResource,
		NewContractResource,
		NewContractTxResource,
	}
}

func (p *EvmProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewContractDataSource,
	}
}

// TODO: Add client instead of version, pass mock backend here
func New(client EvmClient) func() provider.Provider {
	//backends.NewSimulatedBackend
	return func() provider.Provider {
		return &EvmProvider{
			client: client,
		}
	}
}
