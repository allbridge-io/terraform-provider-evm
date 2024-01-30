terraform {
  required_providers {
    evm = {
      source = "terraform.allbridge.io/allbridge/evm"
    }
  }
}

variable "bridge_info" {
  default = {
    1 = {
      chain_precision = 18,
      gas_usage = {
        messenger = 100000,
        wormhole  = 150000,
        bridge    = 200000,
      },
    },
    2 = {
      chain_precision = 18,
      gas_usage = {
        messenger = 50000,
        wormhole  = 100000,
        bridge    = 120000,
      },
    }
  }
}

//////////
// GOERLI
//////////

provider "evm" {
  node_url = "https://ethereum-goerli.publicnode.com"
  alias    = "goerli"
}

resource "evm_random_pk" "deployer_goerli" {
  provider = evm.goerli
}

module "test_tokens_goerli" {
  source = "./test_tokens"
  providers = {
    evm = evm.goerli
  }
  signer = evm_random_pk.deployer_goerli.pk
  tokens = [
    { name = "USD Coin", symbol = "USDC", precision = 18, mint = 1000000000 },
    { name = "Tether USD", symbol = "USDT", precision = 6, mint = 1000000000 },
  ]
}

module "bridge_goerli" {
  source = "./bridge"
  providers = {
    evm = evm.goerli
  }
  signer          = evm_random_pk.deployer_goerli.pk
  chain_id        = 1
  token_addresses = module.test_tokens_goerli.token_addresses
  bridge_info     = var.bridge_info
}

///////
// SEPOLIA
///////

provider "evm" {
  node_url = "https://ethereum-sepolia.publicnode.com"
  alias    = "sepolia"
}

resource "evm_random_pk" "deployer_sepolia" {
  provider = evm.sepolia
}

module "test_tokens_sepolia" {
  source = "./test_tokens"
  providers = {
    evm = evm.sepolia
  }
  signer = evm_random_pk.deployer_sepolia.pk
  tokens = [
    { name = "Tether USD", symbol = "USDT", precision = 6, mint = 1000000000 },
    { name = "Yarros", symbol = "YARO", precision = 12, mint = 1000000000 },
  ]
}

module "bridge_sepolia" {
  source = "./bridge"
  providers = {
    evm = evm.sepolia
  }
  signer          = evm_random_pk.deployer_sepolia.pk
  chain_id        = 2
  token_addresses = module.test_tokens_sepolia.token_addresses
  bridge_info     = var.bridge_info
}

output "sepolia_token_addresses" {
  value = module.test_tokens_sepolia.token_addresses
}

////////
// POST-processing
////////

module "bridge_registration_goerli" {
  source = "./bridge_registration"
  providers = {
    evm = evm.goerli
  }
  signer            = evm_random_pk.deployer_goerli.pk
  chain_id          = 1
  messenger_address = module.bridge_goerli.messenger
  bridge_addresses = {
    1 = module.bridge_goerli.bridge,
    2 = module.bridge_sepolia.bridge,
  }
  token_addresses = {
    1 = module.test_tokens_goerli.token_addresses,
    2 = module.test_tokens_sepolia.token_addresses,
  }
}

module "bridge_registration_sepolia" {
  source = "./bridge_registration"
  providers = {
    evm = evm.sepolia
  }
  signer            = evm_random_pk.deployer_sepolia.pk
  chain_id          = 2
  messenger_address = module.bridge_sepolia.messenger
  bridge_addresses = {
    1 = module.bridge_goerli.bridge,
    2 = module.bridge_sepolia.bridge,
  }
  token_addresses = {
    1 = module.test_tokens_goerli.token_addresses,
    2 = module.test_tokens_sepolia.token_addresses,
  }
}

output "goerli_token_addresses" {
  value = module.test_tokens_goerli.token_addresses
}

output "goerli_bridge" {
  value = module.bridge_goerli.bridge
}