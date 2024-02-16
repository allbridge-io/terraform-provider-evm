resource "evm_contract_tx" "token_transfer" {
  address = evm_contract.test_token.address
  signer  = evm_random_pk.deployer.pk
  method  = "transfer(address,uint256)"
  args = [
    evm_random_pk.token_holder.address,
    20 * pow(10, 18),
  ]
}

output "transfer_tx_id" {
  value = evm_contract_tx.token_transfer.tx_id
}