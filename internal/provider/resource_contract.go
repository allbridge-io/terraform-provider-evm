package provider

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"terraform-provider-evm/internal/utils"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewContractResource() resource.Resource {
	return &contractResource{}
}

type contractResource struct {
	client EvmClient
}

func (*contractResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contract"
}

func (*contractResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deployable and configurable Smart Contract resource.",
		Attributes: map[string]schema.Attribute{
			"artifact": schema.StringAttribute{
				Description: "Content of Hardhat compiled artifact containing ABI and binary.",
				Required:    true,
			},
			"abi": schema.StringAttribute{
				Description: "ABI for the contract interface.",
				Computed:    true,
				Sensitive:   true,
			},
			"bin": schema.StringAttribute{
				Description: "Binary contract code to deploy.",
				Computed:    true,
				Sensitive:   true,
			},
			"signer": schema.StringAttribute{
				Description: "Deploy transaction signer private key.",
				Required:    true,
				Sensitive:   true,
			},
			"address": schema.StringAttribute{
				Description: "Deployed contract address.",
				Computed:    true,
			},
			"constructor_args": schema.ListAttribute{
				Description: "Contract constructor arguments",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *contractResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(EvmClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ethclient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *contractResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan contractModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := plan.parseArtifact()
	if err != nil {
		resp.Diagnostics.AddError("Unexpected error on parsing artifact", err.Error())
		return
	}

	chainID, err := r.client.ChainID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot retrieve chain ID", err.Error())
		return
	}

	privateKey, err := crypto.HexToECDSA(plan.Signer.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error decoding signer to private key", err.Error())
		return
	}

	signerAddress, err := utils.PrivateKeyToAddressString(privateKey)
	if err != nil {
		resp.Diagnostics.AddError("Error calculating signer address", err.Error())
		return
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating signer", err.Error())
		return
	}

	bytecode, err := hex.DecodeString(plan.Bin.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error decoding bytecode", err.Error())
		return
	}

	parsedABI, err := abi.JSON(strings.NewReader(plan.ABI.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Unexpected error on parsing ABI", err.Error())
		return
	}

	args, diags := utils.ParseArguments(ctx, parsedABI.Constructor.Inputs, plan.ConstructorArgs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	address, tx, err := func() (common.Address, *ethTypes.Transaction, error) {
		for {
			address, tx, _, err := bind.DeployContract(auth, parsedABI, bytecode, r.client, args...)
			if err != nil && err.Error() == txpool.ErrReplaceUnderpriced.Error() {
				time.Sleep(1 * time.Second)
				continue
			}
			return address, tx, err
		}
	}()

	if err != nil {
		resp.Diagnostics.AddError("Contract deploy error", "Deployer "+signerAddress+"\n"+err.Error())
		return
	}

	// Wait until transaction is mined
	_, err = bind.WaitMined(ctx, r.client, tx)
	if err != nil {
		resp.Diagnostics.AddError("Error while waiting for transaction to be mined", err.Error())
		return
	}

	plan.Address = types.StringValue(address.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (*contractResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {

}

func (*contractResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var model contractModel
	var stateModel contractModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateModel)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := model.parseArtifact()

	if err != nil {
		resp.Diagnostics.AddError("Unexpected error on parsing artifact", err.Error())
		return
	}

	model.Address = stateModel.Address

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)

}

func (*contractResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

type contractModel struct {
	Artifact        types.String `tfsdk:"artifact"`
	ABI             types.String `tfsdk:"abi"`
	Bin             types.String `tfsdk:"bin"`
	Signer          types.String `tfsdk:"signer"`
	Address         types.String `tfsdk:"address"`
	ConstructorArgs types.List   `tfsdk:"constructor_args"`
}

type artifactModel struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode string          `json:"bytecode"`
}

func (m *contractModel) parseArtifact() error {
	var model artifactModel
	err := json.Unmarshal([]byte(m.Artifact.ValueString()), &model)
	if err != nil {
		return err
	}

	m.ABI = types.StringValue(string(model.ABI))
	m.Bin = types.StringValue(model.Bytecode[2:])

	return nil
}
