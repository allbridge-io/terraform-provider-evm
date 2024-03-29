package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceContract(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "evm_contract" "basic" {
					artifact = file("./testdata/Token.json")
					signer = "` + faucetPk + `"
					constructor_args=["Name","SYM", 1000000000 * pow(10, 18), 18]
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("evm_contract.basic", "address", regexp.MustCompile(`0x[A-Fa-f0-9]{20}`)),
				),
			},
		},
	})
}
