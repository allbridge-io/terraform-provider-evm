variable "signer" {
    type = string
}

variable "chain_id" {
    type = number
}

variable "messenger_address" {
    type = string
}

variable "bridge_addresses" {
    type = map(string)
}

variable "token_addresses" {
    type = map(list(string))
}