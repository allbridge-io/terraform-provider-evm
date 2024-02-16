package utils

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
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

func assertError(t *testing.T, errorActual error, errorExpected error, ifOk func()) {
	if errorExpected != nil {
		if errorActual == nil {
			t.Fatalf("Expected error '%v', got success", errorExpected)
		} else if !errors.Is(errorActual, errorExpected) {
			t.Fatalf("Expected error '%v', got '%v'", errorExpected, errorActual)
		}
	} else {
		if errorActual != nil {
			t.Fatalf("Expected success got error '%v'", errorActual)
		} else {
			ifOk()
		}
	}
}

func TestGenerateFakeABI(t *testing.T) {
	abi, err := GenerateFakeABI(context.TODO(), "transfer", []string{"address", "uint256"})
	if err != nil {
		t.Fatalf("Unexpected error '%v'", err)
	}
	assert.Equal(t, len(abi.Methods), 1)
	assert.Equal(t, abi.Methods["transfer"].Sig, "transfer(address,uint256)")
	assert.Equal(t, abi.Methods["transfer"].ID, []byte{0xa9, 0x05, 0x9c, 0xbb})
}

func TestParseTuple(t *testing.T) {
	var test_data = []struct {
		input  string
		result []string
		err    error
	}{
		{"a,b", []string{"a", "b"}, nil},
		{"a", []string{"a"}, nil},
		{"", []string{}, nil},
		{"(a,b),c", []string{"(a,b)", "c"}, nil},
		{"(a,b,c)", []string{"(a,b,c)"}, nil},
		{"(a,b,c))", nil, ErrUnmatchedBrackets},
		{"a,b,(c)", []string{"a", "b", "(c)"}, nil},
	}

	for _, data := range test_data {
		fmt.Println(data.input)
		result, err := ParseTuple(data.input)
		assertError(t, err, data.err, func() {
			if !reflect.DeepEqual(result, data.result) {
				t.Fatalf("Got '%v', len %v expected '%v', len %v",
					result, len(result), data.result, len(data.result))
			}
		})
	}
}

func TestExtractNameAndTypes(t *testing.T) {
	var test_data = []struct {
		signature string
		name      string
		argTypes  []string
		err       error
	}{
		{"transfer()", "transfer", []string{}, nil},
		{"transfer(address,uint256)", "transfer", []string{"address", "uint256"}, nil},
		{"noArgs", "", nil, ErrUnmatchedBrackets},
		{"weirdBrackets)()", "weirdBrackets)", []string{}, nil},
		{"weirdBrackets)(", "", nil, ErrUnmatchedBrackets},
		{"weirdBrackets(", "", nil, ErrUnmatchedBrackets},
		{"(no, name, only, types)", "", []string{"no", "name", "only", "types"}, nil},
		{"arrays(address[],uint256[3])", "arrays", []string{"address[]", "uint256[3]"}, nil},
		{"tuple((address,uint256))", "tuple", []string{"(address,uint256)"}, nil},
		{"tuple((address,uint256),int256)", "tuple", []string{"(address,uint256)", "int256"}, nil},
	}

	for _, data := range test_data {
		fmt.Println(data.signature)
		name, argTypes, err := ExtractNameAndTypes(context.TODO(), data.signature)
		assertError(t, err, data.err, func() {
			if !reflect.DeepEqual(argTypes, data.argTypes) || name != data.name {
				t.Fatalf("Got '%v' (args '%v', len %v) expected '%v' (args '%v', len %v)",
					name, argTypes, len(argTypes), data.name, data.argTypes, len(data.argTypes))
			}
		})
	}
}

func TestParseArgument(t *testing.T) {

	var test_data = []struct {
		expectedType string
		value        string
		result       any
		err          error
	}{
		{"bool", "0", false, nil},
		{"bool", "false", false, nil},
		{"bool", "true", true, nil},
		{"bool", "", nil, ErrInvalidValueForType},
		{"int", "", nil, ErrInvalidType},
		{"uint", "", nil, ErrInvalidType},
		{"uint256", "hey", nil, ErrInvalidValueForType},
		{"uint255", "1", nil, ErrInvalidType},
		{"int256", "123", big.NewInt(123), nil},
		{"uint256", "123", big.NewInt(123), nil},
		{"int8", "123", int8(123), nil},
		{"uint8", "123", uint8(123), nil},
		{"int16", "1234", int16(1234), nil},
		{"uint16", "1234", uint16(1234), nil},
		{"int32", "-12345", int32(-12345), nil},
		{"uint32", "12345", uint32(12345), nil},
		{"int64", "-123456", int64(-123456), nil},
		{"uint64", "1123456", uint64(1123456), nil},
		{"int224", "1", big.NewInt(1), nil},
		{"bytes", "0011", []byte{0x00, 0x11}, nil},
		{"bytes2", "0011", [2]byte{0x00, 0x11}, nil},
		{"bytes1", "0011", nil, ErrInvalidValueForType},
		{"address", "0", common.Address{}, nil},
		{"address[]", "0", []common.Address{{}}, nil},
		{
			"address",
			"0x11223344556677889900aabbccddeeff11223344",
			common.HexToAddress("0x11223344556677889900aabbccddeeff11223344"),
			nil,
		},
		{"address", "address", nil, ErrInvalidValueForType},
		{
			"bytes32",
			"0x00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
			[32]byte{
				0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
				0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
				0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
				0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			nil,
		},
		{"int256[2]", "123,231", [2]*big.Int{big.NewInt(123), big.NewInt(231)}, nil},
		{"int8[]", "123,-1", []int8{int8(123), int8(-1)}, nil},
	}

	for _, data := range test_data {
		fmt.Println(data.expectedType)
		result, err := parseArgument(context.TODO(), data.expectedType, data.value)
		assertError(t, err, data.err, func() {
			if !reflect.DeepEqual(result, data.result) {
				t.Fatalf("Got '%v' (type '%T') expected '%v' (type '%T')",
					result, result, data.result, data.result)
			}
		})
	}
}
