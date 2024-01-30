package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewContractDataSource is a helper function to simplify the provider implementation.
func NewContractDataSource() datasource.DataSource {
	return &contractDataSource{}
}

type contractDataSource struct {
	client EvmClient
}

type contractDataSourceModel struct {
	Address  types.String `tfsdk:"address"`
	Artifact types.String `tfsdk:"artifact"`
	ABI      types.String `tfsdk:"abi"`
}

// Metadata returns the data source type name.
func (d *contractDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contract"
}

// Schema defines the schema for the data source.
func (d *contractDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "EVM Contract Data Source",

		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				MarkdownDescription: "Contract address",
				Required:            true,
			},
			"artifact": schema.StringAttribute{
				Description: "Content of Hardhat compiled artifact containing ABI and binary.",
				Required:    true,
			},
			"abi": schema.StringAttribute{
				Description: "ABI for the contract interface.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (d *contractDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(EvmClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected EvmClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *contractDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data contractDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := data.parseArtifact()
	if err != nil {
		resp.Diagnostics.AddError("Unexpected error on parsing artifact", err.Error())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type abiModel struct {
	ABI json.RawMessage `json:"abi"`
}

func (m *contractDataSourceModel) parseArtifact() error {
	var model abiModel
	err := json.Unmarshal([]byte(m.Artifact.ValueString()), &model)
	if err != nil {
		return err
	}

	m.ABI = types.StringValue(string(model.ABI))

	return nil
}
