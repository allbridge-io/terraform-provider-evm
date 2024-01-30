terraform {
  required_providers {
    evm = {
      source = "terraform.allbridge.io/allbridge/evm"
    }
  }
}

resource "evm_contract" "test_token" {
  count = length(var.tokens)
  artifact_file = "../bridge/test/Token.sol/Token.json"
  signer        = var.signer
  constructor_args = [
    "\"${var.tokens[count.index].name}\"",
    "\"${var.tokens[count.index].symbol}\"",
    tostring(1000000000 * pow(10, var.tokens[count.index].precision)),
    tostring(var.tokens[count.index].precision),
  ]
}