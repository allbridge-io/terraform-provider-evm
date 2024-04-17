package provider

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-evm/internal/utils"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		MarkdownDescription: "Resource used to deploy new Smart Contracts.",
		Attributes: map[string]schema.Attribute{
			"artifact": schema.StringAttribute{
				MarkdownDescription: "Content of Hardhat compiled artifact containing ABI and binary in JSON format",
				Required:            true,
				Sensitive:           true,
			},
			"signer": schema.StringAttribute{
				MarkdownDescription: "Deploy transaction signer private key (32-byte hex, no `0x` prefix). Can reference `evm_random_pk.pk` resource",
				Required:            true,
				Sensitive:           true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "Deployed contract address, computed after the contract is successfully deployed",
				Computed:            true,
			},
			"constructor_args": schema.ListAttribute{
				MarkdownDescription: "String list of contract constructor arguments. See the list of supported types [here](../../README.md#deployment-and-transaction-args)",
				ElementType:         types.StringType,
				Optional:            true,
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
	r.deployContract(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
}

func (*contractResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {

}

func (r *contractResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.deployContract(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
}

func (*contractResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

type contractModel struct {
	Artifact        types.String `tfsdk:"artifact"`
	Signer          types.String `tfsdk:"signer"`
	Address         types.String `tfsdk:"address"`
	ConstructorArgs types.List   `tfsdk:"constructor_args"`
}

func (r *contractResource) deployContract(ctx context.Context, plan *tfsdk.Plan,
	state *tfsdk.State, respDiags *diag.Diagnostics) {
	var model contractModel

	diags := plan.Get(ctx, &model)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return
	}

	bytecode, err := utils.GetBytecode(model.Artifact.ValueString())
	if err != nil {
		respDiags.AddError("Error parsing bytecode", err.Error())
		return
	}

	argTypes, err := utils.GetConstructorArgTypes(model.Artifact.ValueString())
	if err != nil {
		respDiags.AddError("Error parsing constructor args", err.Error())
		return
	}

	abiJson, err := utils.GetAbi(model.Artifact.ValueString())
	if err != nil {
		respDiags.AddError("Error parsing abi", err.Error())
		return
	}

	chainID, err := r.client.ChainID(ctx)
	if err != nil {
		respDiags.AddError("Cannot retrieve chain ID", err.Error())
		return
	}

	privateKey, err := crypto.HexToECDSA(model.Signer.ValueString())
	if err != nil {
		respDiags.AddError("Error decoding signer to private key", err.Error())
		return
	}

	signerAddress, err := utils.PrivateKeyToAddressString(privateKey)
	if err != nil {
		respDiags.AddError("Error calculating signer address", err.Error())
		return
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		respDiags.AddError("Error creating signer", err.Error())
		return
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiJson))
	if err != nil {
		diags.AddError("Unexpected error on parsing ABI", err.Error())
		return
	}

	args, diags := utils.ParseArguments(ctx, argTypes, model.ConstructorArgs)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return
	}

	address, tx, err := func() (common.Address, *ethTypes.Transaction, error) {
		for {
			address, tx, _, err := bind.DeployContract(auth, parsedABI, bytecode, r.client, args...)
			if err != nil && (err.Error() == txpool.ErrReplaceUnderpriced.Error() || strings.HasPrefix(err.Error(), core.ErrNonceTooLow.Error())) {
				tflog.Info(ctx,
					fmt.Sprintf("Got error '%v' from the node, retrying", err),
				)
				time.Sleep(1 * time.Second)
				continue
			}
			return address, tx, err
		}
	}()

	if err != nil {
		utils.ParseNodeError(signerAddress, err, respDiags)
		if respDiags.HasError() {
			return
		}

		respDiags.AddError(
			"Deploy error",
			fmt.Sprintf("Signer %s\n%v", signerAddress, err),
		)
		return
	}

	// Wait until transaction is mined
	_, err = bind.WaitMined(ctx, r.client, tx)
	if err != nil {
		respDiags.AddError("Error while waiting for transaction to be mined", err.Error())
		return
	}

	model.Address = types.StringValue(address.String())

	respDiags.Append(state.Set(ctx, model)...)
}
