package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	sample_abi = `[{"some":"generic", "json":4}, ["of arbitrary", "structure", 3]]`
	artifact   = `{"abi":` + sample_abi + `}`
)

func TestAccDataSourceContract(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "evm_contract" "basic" {
					address = "0x0"
					artifact= <<-EOT
					` + artifact + `
					EOT
				}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.evm_contract.basic", "abi", sample_abi),
				),
			},
		},
	})
}
