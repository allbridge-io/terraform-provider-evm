# Terraform EVM Provider

Quick way to integrate Ethereum and other EVM blockchains to your deployment. Currently supports generating random deployer accounts, deploying smart contracts and executing transactions on arbitrary smart contracts.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.12+

## Example

Below is a quick example of how you can add the EVM provider to your Terraform script.

```terraform
terraform {
  required_providers {
    evm = {
      source = "terraform.allbridge.io/allbridge/evm"
    }
  }
}

// Declare provider
provider "evm" {
  node_url = "https://ethereum-sepolia.publicnode.com"
}

// Generate random accounts
resource "evm_random_pk" "deployer" {}
resource "evm_random_pk" "token_holder" {}

// Deploy smart contract
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

// Execute transaction
resource "evm_contract_tx" "token_transfer" {
  address = evm_contract.test_token.address
  signer  = evm_random_pk.deployer.pk
  method  = "transfer(address,uint256)"
  args = [
    evm_random_pk.token_holder.address,
    20 * pow(10, 18),
  ]
}

// Use outputs to print computed values
output "deployer_address" {
  value = evm_random_pk.deployer.address
}
output "holder_address" {
  value = evm_random_pk.token_holder.address
}
output "token_contract_address" {
  value = evm_contract.test_token.address
}
output "transfer_tx_id" {
  value = evm_contract_tx.token_transfer.tx_id
}
```

## Deployment and transaction args

Arguments to contract calls should be supplied as the list of strings. EVM provider will interpret constructor arguments from the contract ABI or provided method signature to infer their types.

Following types are currently supported:
- `bool` supporting values `true`/`false` or `1`/`0`
- signed `int8` to `int256` and unsigned `uint8` to `uint256`
- `address` with empty string, `0x` or `0` as shorthands for zero address
- unsized `bytes` and `bytes1` to `bytes32` byte arrays, values should be specified as hex strings (e.g. `0x0102` for `bytes2` value)
- fixed size or dynamic single-dimension arrays of all types listed above (e.g. `address[]` or `int32[4]`). Values for arrays should be comma-separated (e.g. `1,2` for `int32[2]` or `one,two` for `string[]`)

Not yet supported types:
- multi-dimensional arrays
- user structs (encoded as tuples)

## Documentation

Documentation is generated with
[tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). Generated
files are in `docs/` and should not be updated manually. They are derived from:

- Schema `Description` fields in the provider Go code
- [examples/](./examples)
- [templates/](./templates)

Use `go generate` to update generated docs.

## Local Development

To build the EVM provider locally you will need [Go](https://go.dev/doc/install) version 1.20+ installed. To build the provider use:

```
go install
```

EVM provider will be installed in the directory named by the GOBIN environment
variable, which defaults to $GOPATH/bin or $HOME/go/bin if the GOPATH
environment variable is not set. Executables in $GOROOT
are installed in $GOROOT/bin or $GOTOOLDIR instead of $GOBIN.

Then create a `.terraformrc` file in your home folder to override EVM provider to use your locally built installation.

```terraform
provider_installation {

  dev_overrides {
      "terraform.allbridge.io/allbridge/evm" = "/path/to/your/go/bin/folder"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

### Running Tests

You do not need an interface to an actual blockchain to run tests, they are executed on a simulated backend.

In order to run the full suite of Acceptance tests, run:

```
make testacc
```