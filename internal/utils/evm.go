package utils

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func PrivateKeyToAddressString(privateKey *ecdsa.PrivateKey) (string, error) {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("Unexpected public key casting error")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	return address, nil
}

func ParseArguments(ctx context.Context, expected abi.Arguments, provided basetypes.ListValue) ([]interface{}, diag.Diagnostics) {

	tflog.Info(ctx, "Parsing arguments")
	lenArguments := len(provided.Elements())
	lenABIArguments := len(expected)
	if lenArguments != lenABIArguments {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Invalid arguments",
				fmt.Sprintf("Found %d arguments, expexted %d from ABI", lenArguments, lenABIArguments),
			),
		}
	}

	args := make([]interface{}, lenArguments)
	elements := make([]types.String, 0, lenArguments)
	diags := provided.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, diags
	}

	for i, arg := range elements {
		argABI := expected[i]
		expectedType := argABI.Type.GetType()

		valueInterface := reflect.New(expectedType).Interface()

		tflog.Info(ctx,
			fmt.Sprintf("Adding argument %v with expected type %v ABI type %v", arg, expectedType, argABI.Type.T),
		)

		stringValue := arg.ValueString()

		if strings.HasPrefix(stringValue, "\"0x") && (argABI.Type.T == abi.FixedBytesTy || argABI.Type.T == abi.BytesTy) {
			// This is a special case when hex needs to be parsed to a list
			value, err := hex.DecodeString(strings.Trim(stringValue, "\"")[2:])
			if err != nil {
				return nil, diag.Diagnostics{
					diag.NewErrorDiagnostic(
						"Cannot decode hex for array/slice parameter",
						fmt.Sprintf("Value :%s", stringValue),
					),
				}
			}
			tflog.Info(ctx, fmt.Sprintf("Decoded value %v", value))
			// Convert type so that mershal will encode it as an array
			valueInts := make([]int, len(value))
			for i, element := range value {
				valueInts[i] = int(element)
			}
			remarshaled, err := json.Marshal(valueInts)
			if err != nil {
				return nil, diag.Diagnostics{
					diag.NewErrorDiagnostic(
						"Cannot reencode hex for array/slice parameter",
						fmt.Sprintf("Value :%v", value),
					),
				}
			}
			stringValue = string(remarshaled)
			tflog.Info(ctx,
				fmt.Sprintf("Remarshaled into %v", stringValue),
			)
		}

		json.Unmarshal([]byte(stringValue), valueInterface)

		tflog.Info(ctx, fmt.Sprintf("Unmarshalled into %v", valueInterface))

		args[i] = valueInterface
	}

	tflog.Info(ctx, fmt.Sprintf("Prepared arguments: %v", args))

	return args, diags
}
