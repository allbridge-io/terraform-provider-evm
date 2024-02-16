package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceContractTx(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "evm_contract" "basic" {
					artifact = file("./testdata/Token.json")
					signer = "` + faucetPk + `"
					constructor_args=["Name","SYM", 1000000000 * pow(10, 18), 18]
				}
				
				resource "evm_contract_tx" "token_transfer" {
					address = evm_contract.basic.address
					signer = "` + faucetPk + `"
					method = "transfer(address,uint256)"
					args=["0x000000000000000000000000000000000000dead", 10 * pow(10, 18)]
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("evm_contract_tx.token_transfer", "tx_id", regexp.MustCompile(`0x[A-Fa-f0-9]{20}`)),
				),
			},
		},
	})
}
