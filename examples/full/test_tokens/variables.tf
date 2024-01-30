variable "tokens" {
    type = list(object({
        name = string
        symbol = string
        precision = number
        mint = number
    }))
}

variable "signer" {
    type = string
}