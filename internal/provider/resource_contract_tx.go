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
	"github.com/ethereum/go-ethereum/core/txpool"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewContractTxResource() resource.Resource {
	return &contractTxResource{}
}

type contractTxResource struct {
	client *ethclient.Client
}

func (*contractTxResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contract_tx"
}

func (*contractTxResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deployable and configurable smart contract resource.",
		Attributes: map[string]schema.Attribute{
			"abi": schema.StringAttribute{
				Description: "ABI for the contract interface.",
				Required:    true,
				Sensitive:   true,
			},
			"address": schema.StringAttribute{
				Description: "Contract address to call.",
				Required:    true,
			},
			"signer": schema.StringAttribute{
				Description: "Deploy transaction signer private key.",
				Required:    true,
				Sensitive:   true,
			},
			"method": schema.StringAttribute{
				Description: "Contract method to call.",
				Required:    true,
			},
			"args": schema.ListAttribute{
				Description: "Method call arguments.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"tx_id": schema.StringAttribute{
				Description: "Transaction id of submitted transaction.",
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

	client, ok := req.ProviderData.(*ethclient.Client)

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
	var plan contractTxModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tx_id, diags := r.prepareAndSendTransaction(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.TxId = types.StringValue(tx_id)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (*contractTxResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {

}

func (r *contractTxResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan contractTxModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tx_id, diags := r.prepareAndSendTransaction(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.TxId = types.StringValue(tx_id)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (*contractTxResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

type contractTxModel struct {
	ABI     types.String `tfsdk:"abi"`
	Signer  types.String `tfsdk:"signer"`
	Address types.String `tfsdk:"address"`
	Method  types.String `tfsdk:"method"`
	Args    types.List   `tfsdk:"args"`
	TxId    types.String `tfsdk:"tx_id"`
}

func (r *contractTxResource) prepareAndSendTransaction(ctx context.Context, plan contractTxModel) (string, diag.Diagnostics) {

	var diags diag.Diagnostics

	chainID, err := r.client.ChainID(ctx)
	if err != nil {
		diags.AddError("Cannot retrieve chain ID", err.Error())
		return "", diags
	}

	privateKey, err := crypto.HexToECDSA(plan.Signer.ValueString())
	if err != nil {
		diags.AddError("Error decoding signer to private key", err.Error())
		return "", diags
	}

	signerAddress, err := utils.PrivateKeyToAddressString(privateKey)
	if err != nil {
		diags.AddError("Error calculating signer address", err.Error())
		return "", diags
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		diags.AddError("Error creating signer", err.Error())
		return "", diags
	}

	parsedABI, err := abi.JSON(strings.NewReader(plan.ABI.ValueString()))
	if err != nil {
		diags.AddError("Unexpected error on parsing ABI", err.Error())
		return "", diags
	}

	method := plan.Method.ValueString()

	args, parseDiags := utils.ParseArguments(ctx, parsedABI.Methods[method].Inputs, plan.Args)
	diags.Append(parseDiags...)
	if diags.HasError() {
		return "", diags
	}

	contractAddress := common.HexToAddress(plan.Address.ValueString())

	c := bind.NewBoundContract(contractAddress, parsedABI, r.client, r.client, r.client)

	tx, err := func() (*ethTypes.Transaction, error) {
		for {
			tx, err := c.Transact(auth, method, args...)
			if err != nil && err.Error() == txpool.ErrReplaceUnderpriced.Error() {
				time.Sleep(1 * time.Second)
				continue
			}
			return tx, err
		}
	}()

	if err != nil {
		diags.AddError("Transaction error", "Signer "+signerAddress+"\n"+err.Error())
		return "", diags
	}

	// Wait until transaction is mined
	_, err = bind.WaitMined(ctx, r.client, tx)
	if err != nil {
		diags.AddError("Error while waiting for transaction to be mined", err.Error())
		return "", diags
	}

	return tx.Hash().String(), diags
}
