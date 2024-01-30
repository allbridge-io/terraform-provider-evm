terraform {
  required_providers {
    evm = {
      source = "terraform.allbridge.io/allbridge/evm"
    }
  }
}

data "evm_contract" "bridge" {
  artifact_file = "../bridge/Bridge.sol/Bridge.json"
  address = var.bridge_addresses[tostring(var.chain_id)]
}

data "evm_contract" "messenger" {
  artifact_file = "../bridge/Messenger.sol/Messenger.json"
  address = var.messenger_address
}

// Set other chain ids
resource "evm_contract_tx" "other_chains" {
  address  = data.evm_contract.messenger.address
  abi      = data.evm_contract.messenger.abi
  signer   = var.signer
  method   = "setOtherChainIds"
  args = [
    "[${join(",", [for i in range(32) : i != var.chain_id && contains(keys(var.bridge_addresses), tostring(i))? 1: 0])}]",
  ]
}

// Set other bridge addresses
resource "evm_contract_tx" "other_bridge" {
  for_each = { for k, v in var.bridge_addresses : k => v if k != tostring(var.chain_id) }
  address  = data.evm_contract.bridge.address
  abi      = data.evm_contract.bridge.abi
  signer   = var.signer
  method   = "registerBridge"
  args = [
    each.key,
    "\"0x000000000000000000000000${substr(each.value, 2, -1)}\"",
  ]
}

// Set other tokens
locals {
  other_tokens = flatten([for chain, token_list in var.token_addresses : [for addr in token_list : {chain=chain, addr=addr}] if chain != tostring(var.chain_id)])
}

resource "evm_contract_tx" "other_bridge_token" {
  count   = length(local.other_tokens)
  address = data.evm_contract.bridge.address
  abi     = data.evm_contract.bridge.abi
  signer  = var.signer
  method  = "addBridgeToken"
  args = [
    local.other_tokens[count.index]["chain"],
    "\"0x000000000000000000000000${substr(local.other_tokens[count.index]["addr"], 2, -1)}\"",
  ]
}
