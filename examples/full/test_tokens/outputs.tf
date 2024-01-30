output "token_addresses" {
    value = evm_contract.test_token[*].address
}