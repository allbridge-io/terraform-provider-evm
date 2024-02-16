resource "evm_random_pk" "account" {
  // No input required, all fields are computed
}

output "account_address" {
  value = evm_random_pk.account.address
}