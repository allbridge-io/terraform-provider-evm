package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "evm_random_pk" "basic" { 
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("evm_random_pk.basic", "pk", regexp.MustCompile(`[A-Fa-f0-9]{64}`)),
					resource.TestMatchResourceAttr("evm_random_pk.basic", "pub_key", regexp.MustCompile(`[A-Fa-f0-9]{128}`)),
					resource.TestMatchResourceAttr("evm_random_pk.basic", "address", regexp.MustCompile(`0x[A-Fa-f0-9]{20}`)),
				),
			},
		},
	})
}
