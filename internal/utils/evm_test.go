package utils

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestPrivateKeyToAddressString(t *testing.T) {
	pk := "75ad181fe502bfc2031ec1ee5c6d34c65171cc44bca49ddf59940b1766420969"
	address := "0x76116eb4BcE815c280c471620286e606D51Eea73"
	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		t.Fatalf("Error converting string to private key: %v", err)
	}
	calculatedAddress, err := PrivateKeyToAddressString(privateKey)
	if err != nil {
		t.Fatalf("Error calculating address: %v", err)
	}
	if address != calculatedAddress {
		t.Fatalf("Expected address %v, got %v", address, calculatedAddress)
	}
}
