resource "evm_contract" "test_token" {
  artifact = file("./internal/provider/testdata/Token.json")
  signer   = evm_random_pk.deployer.pk
  constructor_args = [
    "Test1",
    "TST",
    1000000000 * pow(10, 18),
    18,
  ]
}

output "token_contract_address" {
  value = evm_contract.test_token.address
}