package utils

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func ParseNodeError(signer string, err error, diags *diag.Diagnostics) {
	regexpInsufficientGas := regexp.MustCompile(`insufficient\sfunds\sfor\sgas\s\*\sprice\s\+\svalue:\sbalance\s(\d+),\stx\scost\s(\d+),\sovershot\s(\d+)`)
	matches := regexpInsufficientGas.FindStringSubmatch(err.Error())
	if matches != nil {
		diags.AddError(
			"Insufficient gas error",
			fmt.Sprintf(
				"Transaction cost %v ETH, have %v ETH\nFund '%s' at least %v ETH",
				WeiToEther(matches[2]),
				WeiToEther(matches[1]),
				signer,
				WeiToEther(matches[3]),
			),
		)
		return
	}
}
