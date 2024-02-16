package utils

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func PrivateKeyToAddressString(privateKey *ecdsa.PrivateKey) (string, error) {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("unexpected public key casting error")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	return address, nil
}

func WeiToEther(wei string) float64 {
	weiAsFloat, ok := new(big.Float).SetString(wei)
	if !ok {
		weiAsFloat = big.NewFloat(0.0)
	}
	result, _ := new(big.Float).Quo(weiAsFloat, big.NewFloat(params.Ether)).Float64()
	return result
}

func ParseArguments(ctx context.Context, argTypes []string, argValues basetypes.ListValue) ([]interface{}, diag.Diagnostics) {

	tflog.Info(ctx, "Parsing arguments")
	lenArguments := len(argValues.Elements())
	lenABIArguments := len(argTypes)
	if lenArguments != lenABIArguments {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Invalid arguments",
				fmt.Sprintf("Found %d arguments, expected %d from ABI", lenArguments, lenABIArguments),
			),
		}
	}

	args := make([]interface{}, lenArguments)
	elements := make([]types.String, 0, lenArguments)
	diags := argValues.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, diags
	}

	for i, arg := range elements {
		var err error
		args[i], err = parseArgument(ctx, argTypes[i], arg.ValueString())
		if err != nil {
			return nil, diag.Diagnostics{
				diag.NewErrorDiagnostic("Invalid argument", fmt.Sprintf("Details: %v", err.Error())),
			}
		}
	}

	tflog.Info(ctx, fmt.Sprintf("Prepared arguments: %v", args))

	return args, diags
}

func GenerateFakeABI(ctx context.Context, name string, argTypes []string) (abi.ABI, error) {

	type FakeInput struct {
		Type string
	}

	type FakeABIItem struct {
		Inputs []FakeInput
		Type   string
		Name   string
	}

	// Create fake ABI JSON and make ABI unmarshal itself
	inputs := make([]FakeInput, len(argTypes))
	for i, typeName := range argTypes {
		inputs[i] = FakeInput{typeName}
	}

	var fakeABI []FakeABIItem
	if name == "" {
		fakeABI = []FakeABIItem{{inputs, "constructor", ""}}
	} else {
		fakeABI = []FakeABIItem{{inputs, "function", name}}
	}

	fakeJSON, err := json.Marshal(fakeABI)
	fmt.Println(string(fakeJSON))
	if err != nil {
		return abi.ABI{}, err
	}

	return abi.JSON(bytes.NewReader(fakeJSON))
}

func ParseTuple(tuple string) ([]string, error) {

	// Optimistically count commas as a number of elements to allocate
	result := make([]string, 0, strings.Count(tuple, ",")+1)
	brackets := 0
	elementStart := 0
	for i, ch := range tuple {
		if ch == '(' {
			brackets += 1
		} else if ch == ')' {
			brackets -= 1
		}

		if ch == ',' || i == len(tuple)-1 {
			if brackets == 0 {
				lastIndex := i
				if i == len(tuple)-1 {
					lastIndex += 1
				}
				trimmedElement := strings.TrimSpace(tuple[elementStart:lastIndex])
				if len(trimmedElement) != 0 {
					result = append(result, trimmedElement)
				}
				elementStart = i + 1
			}
		}
	}

	if brackets != 0 {
		return nil, errors.Join(ErrUnmatchedBrackets, fmt.Errorf("'%v'", tuple))
	} else {
		return result, nil
	}

}

func ExtractNameAndTypes(ctx context.Context, methodSignature string) (string, []string, error) {
	firstBracket := strings.Index(methodSignature, "(")
	lastBracket := strings.LastIndex(methodSignature, ")")
	if firstBracket == -1 || lastBracket == -1 || lastBracket <= firstBracket {
		return "", nil, errors.Join(ErrUnmatchedBrackets, fmt.Errorf("'%v'", methodSignature))
	}
	name := strings.TrimSpace(methodSignature[:firstBracket])
	args := methodSignature[firstBracket+1 : lastBracket]
	returnArgs, err := ParseTuple(args)
	if err != nil {
		return "", nil, err
	}
	return name, returnArgs, nil
}

var (
	// typeRegex parses the abi sub types
	typeRegex       = regexp.MustCompile("([a-zA-Z]+)(([0-9]+)(x([0-9]+))?)?")
	addressRegex    = regexp.MustCompile("0x[a-fA-F0-9]{40}")
	sliceArrayRegex = regexp.MustCompile(`((?:[a-zA-Z]+)(?:(?:[0-9]+)(?:x(?:[0-9]+))?)?)\[([0-9]*)\]`)
)

var (
	ErrInvalidType         = errors.New("invalid type")
	ErrInvalidValueForType = errors.New("invalid value")
	ErrUnmatchedBrackets   = errors.New("unmatched brackets")
)

func parseArgument(ctx context.Context, expectedType string, value string) (interface{}, error) {
	tflog.Info(ctx,
		fmt.Sprintf("Adding argument '%v' with expected type '%v'", value, expectedType),
	)

	sliceArrayMatch := sliceArrayRegex.FindStringSubmatch(expectedType)
	if sliceArrayMatch != nil {
		baseType := sliceArrayMatch[1]
		length := 0
		if sliceArrayMatch[2] == "" {
			tflog.Info(ctx,
				fmt.Sprintf("Found slice type '%v'", baseType),
			)
		} else {
			tflog.Info(ctx,
				fmt.Sprintf("Found array type '%v' length '%v'", baseType, sliceArrayMatch[2]),
			)
			var err error
			length, err = strconv.Atoi(sliceArrayMatch[2])
			if err != nil {
				return nil, ErrInvalidType
			}
		}
		// Split value by comma and parse values recursively
		values := strings.Split(value, ",")
		if values == nil {
			return nil, ErrInvalidValueForType
		}
		var result reflect.Value = reflect.ValueOf(0)
		for _, element := range values {
			resItem, err := parseArgument(ctx, baseType, element)
			if err != nil {
				return nil, err
			}
			if result.IsZero() {
				result = reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(resItem)), 0, len(values))
			}

			tflog.Info(ctx,
				fmt.Sprintf("Result type '%T', item type '%T'", result.Interface(), resItem),
			)

			result = reflect.Append(result, reflect.ValueOf(resItem))

		}
		if length == 0 {
			return result.Interface(), nil
		} else {
			return sliceToArray(result.Interface(), length)
		}

	}

	matches := typeRegex.FindAllStringSubmatch(expectedType, -1)
	if len(matches) == 0 {
		return nil, errors.Join(ErrInvalidType, fmt.Errorf("'%v'", expectedType))
	}
	parsedType := matches[0]

	tflog.Info(ctx,
		fmt.Sprintf("Type after parsing '%v'", parsedType),
	)

	var varSize int
	if len(parsedType[3]) > 0 {
		var err error
		varSize, err = strconv.Atoi(parsedType[2])
		if err != nil {
			return nil, nil
		}
	} else {
		if parsedType[0] == "uint" || parsedType[0] == "int" {
			// this should fail because it means that there's something wrong with
			// the abi type (the compiler should always format it to the size...always)
			return nil, errors.Join(ErrInvalidType, fmt.Errorf("integer type '%v' should include size", expectedType))
		}
	}

	switch parsedType[1] {
	case "bool":
		switch value {
		case "0", "false":
			return false, nil
		case "1", "true":
			return true, nil
		default:
			return false, errors.Join(ErrInvalidValueForType, fmt.Errorf("cannot case `%v` to bool", value))
		}
	case "string":
		return value, nil
	case "uint":
		return parseInteger(value, false, varSize)
	case "int":
		return parseInteger(value, true, varSize)
	case "bytes":
		return parseBytes(value, varSize)
	case "address":
		if value == "0x" || value == "" || value == "0" {
			// Return empty address
			return common.Address{}, nil
		}
		// Else validate the address
		if !addressRegex.MatchString(value) {
			return nil, errors.Join(ErrInvalidValueForType, fmt.Errorf("`%v` does not match address format", value))
		}
		return common.HexToAddress(value), nil
	}
	return nil, errors.Join(ErrInvalidType, fmt.Errorf("'%v'", expectedType))
}

func parseInteger(value string, isSigned bool, size int) (interface{}, error) {
	if size%8 != 0 {
		return nil, ErrInvalidType
	}
	if size == 8 || size == 16 || size == 32 || size == 64 {
		// Regular signed/unsigned int
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, errors.Join(ErrInvalidValueForType, fmt.Errorf("'%v' cannot parse as 'int'", value))
		}
		if isSigned {
			switch size {
			case 8:
				return int8(n), nil
			case 16:
				return int16(n), nil
			case 32:
				return int32(n), nil
			case 64:
				return int64(n), nil
			}
		} else {
			switch size {
			case 8:
				return uint8(n), nil
			case 16:
				return uint16(n), nil
			case 32:
				return uint32(n), nil
			case 64:
				return uint64(n), nil
			}
		}
		return nil, ErrInvalidType
	} else {
		// Big int
		n := new(big.Int)
		n, ok := n.SetString(value, 10)
		if !ok {
			return nil, errors.Join(ErrInvalidValueForType, fmt.Errorf("'%v' cannot parse as 'big.Int'", value))
		}
		return n, nil
	}
}

func parseBytes(value string, length int) (any, error) {
	bytesAsSlice := common.FromHex(value)
	if length == 0 {
		return bytesAsSlice, nil
	}
	if length < 1 || length > 32 {
		return nil, errors.Join(ErrInvalidType, fmt.Errorf("length %v' not supported", length))
	}
	if len(bytesAsSlice) != length {
		return nil, errors.Join(ErrInvalidValueForType, fmt.Errorf("expected data of length %v, got %v", length, len(bytesAsSlice)))
	}
	return sliceToArray(bytesAsSlice, length)
}

func sliceToArray(slice any, length int) (any, error) {
	sliceType := reflect.TypeOf(slice)
	if sliceType.Kind() != reflect.Slice {
		return nil, ErrInvalidValueForType
	}
	elementType := sliceType.Elem()
	result := reflect.New(reflect.ArrayOf(length, elementType)).Elem()
	sliceValue := reflect.ValueOf(slice)
	for i := 0; i < sliceValue.Len(); i++ {
		element := sliceValue.Index(i)
		result.Index(i).Set(element)
	}
	return result.Interface(), nil
}
