package provider

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-evm/internal/utils"
	"time"

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

func NewContractTxResource() resource.Resource {
	return &contractTxResource{}
}

type contractTxResource struct {
	client EvmClient
}

func (*contractTxResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contract_tx"
}

func (*contractTxResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource for triggering transactions on deployed smart contracts.",
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				Description: "Blockchain address of the contract to execute transaction on (20-byte hex with `0x` prefix)",
				Required:    true,
			},
			"signer": schema.StringAttribute{
				Description: "Deploy transaction signer private key (32-byte hex, no `0x` prefix). Can reference `evm_random_pk.pk` resource",
				Required:    true,
				Sensitive:   true,
			},
			"method": schema.StringAttribute{
				Description: "Contract function to execute, specified as a function name with comma-separated parameter types in brackets (e.g. `transfer(address,uint256)`), see the list of supported types [here](../../README.md#deployment-and-transaction-args)",
				Required:    true,
			},
			"args": schema.ListAttribute{
				Description: "String list of contract function arguments. See the list of supported types [here](../../README.md#deployment-and-transaction-args)",
				ElementType: types.StringType,
				Optional:    true,
			},
			"tx_id": schema.StringAttribute{
				Description: "Transaction id of submitted transaction, populated after transaction is executed.",
				Computed:    true,
			},
		},
	}
}

func (r *contractTxResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *contractTxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.prepareAndSendTransaction(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
}

func (*contractTxResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {

}

func (r *contractTxResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.prepareAndSendTransaction(ctx, &req.Plan, &resp.State, &resp.Diagnostics)
}

func (*contractTxResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

type contractTxModel struct {
	Signer  types.String `tfsdk:"signer"`
	Address types.String `tfsdk:"address"`
	Method  types.String `tfsdk:"method"`
	Args    types.List   `tfsdk:"args"`
	TxId    types.String `tfsdk:"tx_id"`
}

func (r *contractTxResource) prepareAndSendTransaction(ctx context.Context, plan *tfsdk.Plan,
	state *tfsdk.State, respDiags *diag.Diagnostics) {

	var model contractTxModel

	diags := plan.Get(ctx, &model)
	respDiags.Append(diags...)
	if respDiags.HasError() {
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

	// Method signature in transfer(address,uint256) format
	methodSignature := model.Method.ValueString()
	methodName, expectedTypes, err := utils.ExtractNameAndTypes(ctx, methodSignature)
	if err != nil {
		respDiags.AddError("Unexpected error on parsing method signature", err.Error())
		return
	}

	fakeABI, err := utils.GenerateFakeABI(ctx, methodName, expectedTypes)
	if err != nil {
		respDiags.AddError("Unexpected error on parsing method signature", err.Error())
		return
	}

	args, parseDiags := utils.ParseArguments(ctx, expectedTypes, model.Args)
	respDiags.Append(parseDiags...)
	if respDiags.HasError() {
		return
	}

	contractAddress := common.HexToAddress(model.Address.ValueString())

	c := bind.NewBoundContract(contractAddress, fakeABI, r.client, r.client, r.client)

	tx, err := func() (*ethTypes.Transaction, error) {
		for {
			tx, err := c.Transact(auth, methodName, args...)
			if err != nil && (err.Error() == txpool.ErrReplaceUnderpriced.Error() || strings.HasPrefix(err.Error(), core.ErrNonceTooLow.Error())) {
				tflog.Info(ctx,
					fmt.Sprintf("Got error '%v' from the node, retrying", err),
				)
				time.Sleep(1 * time.Second)
				continue
			}
			return tx, err
		}
	}()

	if err != nil {
		utils.ParseNodeError(signerAddress, err, respDiags)
		if respDiags.HasError() {
			return
		}

		respDiags.AddError(
			"Transaction error",
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

	model.TxId = types.StringValue(tx.Hash().String())

	respDiags.Append(state.Set(ctx, model)...)
}
