variable "signer" {
    type = string
}

variable "chain_id" {
    type = number
}

variable "token_addresses" {
    type = list(string)
}

variable "bridge_info" {
    type = map(object({
        chain_precision = number,
        gas_usage = object({
            messenger = number,
            wormhole = number,
            bridge = number,
        })
    }))
}