output "gas_oracle" {
  value = evm_contract.gas_oracle.address
}

output "messenger" {
  value = evm_contract.messenger.address
}

output "bridge" {
  value = evm_contract.bridge.address
}

output "pools" {
  value = evm_contract.pool[*].address
}

output "primary_validator_pk" {
  value     = evm_random_pk.primary_validator.pk
  sensitive = true
}

output "secondary_validator_pk" {
  value     = evm_random_pk.secondary_validator.pk
  sensitive = true
}