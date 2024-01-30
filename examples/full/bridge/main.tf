terraform {
  required_providers {
    evm = {
      source = "terraform.allbridge.io/allbridge/evm"
    }
  }
}

variable "wormhole_messenger_address" {
  type    = string
  default = "0x0000000000000000000000000000000000000000"
}
variable "pool_a_value" {
  type    = string
  default = "20"
}
variable "pool_fee_share_bp" {
  type    = string
  default = "15"
}
variable "pool_balance_ratio_min_bp" {
  type    = string
  default = "1000"
}
variable "pool_lp_token_name" {
  type    = string
  default = "Yaro Pool LP Token"
}
variable "pool_lp_token_symbol" {
  type    = string
  default = "LP-YARO"
}

resource "evm_random_pk" "primary_validator" {
}

resource "evm_random_pk" "secondary_validator" {
}

locals {
  chain_precision = var.bridge_info[tostring(var.chain_id)].chain_precision
}

resource "evm_contract" "gas_oracle" {
  artifact_file    = "../bridge/GasOracle.sol/GasOracle.json"
  signer           = var.signer
  constructor_args = [var.chain_id, local.chain_precision]
}

resource "evm_contract" "messenger" {
  artifact_file = "../bridge/Messenger.sol/Messenger.json"
  signer        = var.signer
  constructor_args = [
    var.chain_id,
    "[${join(",", [for i in range(32) : i != var.chain_id && contains(keys(var.bridge_info), tostring(i))? 1: 0])}]",
    "\"${evm_contract.gas_oracle.address}\"",
    "\"${evm_random_pk.primary_validator.address}\"",
    "[\"${evm_random_pk.secondary_validator.address}\"]"
  ]
}

resource "evm_contract" "bridge" {
  artifact_file = "../bridge/Bridge.sol/Bridge.json"
  signer        = var.signer
  constructor_args = [
    var.chain_id,
    local.chain_precision,
    "\"${evm_contract.messenger.address}\"",
    "\"${var.wormhole_messenger_address}\"",
    "\"${evm_contract.gas_oracle.address}\""
  ]
}

resource "evm_contract" "pool" {
  count = length(var.token_addresses)
  artifact_file = "../bridge/Pool.sol/Pool.json"
  signer        = var.signer
  constructor_args = [
    "\"${evm_contract.bridge.address}\"",
    var.pool_a_value,
    "\"${var.token_addresses[count.index]}\"",
    var.pool_fee_share_bp,
    var.pool_balance_ratio_min_bp,
    "\"${var.pool_lp_token_name}\"",
    "\"${var.pool_lp_token_symbol}\"",
  ]
}
/*
resource "evm_contract_tx" "token_allowance" {
  address = evm_contract.test_token_1.address
  abi     = evm_contract.test_token_1.abi
  signer  = evm_random_pk.deployer.pk
  method  = "approve"
  args = [
    "\"${evm_contract.pool.address}\"",
    tostring(pow(2, 256) - 1)
  ]
}

resource "evm_contract_tx" "deposit_liquidity" {
  address = evm_contract.pool.address
  abi = evm_contract.pool.abi
  signer = evm_random_pk.deployer.pk
  method = "deposit"
  args = [
    tostring(tonumber(var.deposit_amount) * pow(10, var.test_token_precision))
  ]
  depends_on = [ evm_contract_tx.token_allowance ]
}
*/
resource "evm_contract_tx" "add_pool" {
  count = length(var.token_addresses)
  address = evm_contract.bridge.address
  abi = evm_contract.bridge.abi
  signer = var.signer
  method = "addPool"
  args = [
    "\"${evm_contract.pool[count.index].address}\"",
    "\"0x000000000000000000000000${substr(var.token_addresses[count.index], 2, -1)}\"",
  ]
}

resource "evm_contract_tx" "bridge_gas_usage" {
  for_each = {for k, v in var.bridge_info : k => v if k != tostring(var.chain_id)}
  address = evm_contract.bridge.address
  abi = evm_contract.bridge.abi
  signer = var.signer
  method = "setGasUsage"
  args = [
    each.key,
    tostring(each.value.gas_usage.bridge),
  ]
}

resource "evm_contract_tx" "messenger_gas_usage" {
  for_each = {for k, v in var.bridge_info : k => v if k != tostring(var.chain_id)}
  address = evm_contract.messenger.address
  abi = evm_contract.messenger.abi
  signer = var.signer
  method = "setGasUsage"
  args = [
    each.key,
    tostring(each.value.gas_usage.messenger),
  ]
}