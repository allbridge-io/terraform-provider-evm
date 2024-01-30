package provider

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewRandomPkResource() resource.Resource {
	return &pkResource{}
}

type pkResource struct{}

func (*pkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_random_pk"
}

func (*pkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Random private key for signing transactions.",
		Attributes: map[string]schema.Attribute{
			"pk": schema.StringAttribute{
				Description: "The generated random private key.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pub_key": schema.StringAttribute{
				Description: "Public key calculated from the provate key.",
				Computed:    true,
			},
			"address": schema.StringAttribute{
				Description: "EVM address calculated from the provate key.",
				Computed:    true,
			},
		},
	}
}

// Create implements resource.Resource.
func (*pkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pkModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected error on private key generation",
			err.Error(),
		)
		return
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected error casting public key to ECDSA",
			err.Error(),
		)
		return
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	plan.PK = types.StringValue(hexutil.Encode(privateKeyBytes)[2:])
	plan.PubKey = types.StringValue(hexutil.Encode(publicKeyBytes)[2:])
	plan.Address = types.StringValue(address)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read does not need to perform any operations as the state in ReadResourceResponse is already populated.
func (*pkResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {

}

// Update ensures the plan value is copied to the state to complete the update.
func (*pkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model pkModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// Delete does not need to explicitly call resp.State.RemoveResource() as this is automatically handled by the
// [framework](https://github.com/hashicorp/terraform-plugin-framework/pull/301).
func (*pkResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

type pkModel struct {
	PK      types.String `tfsdk:"pk"`
	PubKey  types.String `tfsdk:"pub_key"`
	Address types.String `tfsdk:"address"`
}
