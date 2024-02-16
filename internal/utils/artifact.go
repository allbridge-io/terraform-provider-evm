package utils

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/tidwall/gjson"
)

var (
	ErrArtifactFieldNotFound    = errors.New("field not found")
	ErrArtifactWrongFieldFormat = errors.New("wrong field format")
)

func GetBytecode(artifactJson string) ([]byte, error) {
	value := gjson.Get(artifactJson, "bytecode")
	if !value.Exists() {
		return nil, ErrArtifactFieldNotFound
	}

	valueStr := value.String()
	if !strings.HasPrefix(valueStr, "0x") {
		return nil, ErrArtifactWrongFieldFormat
	}

	bytecode, err := hex.DecodeString(valueStr[2:])
	if err != nil {
		return nil, errors.Join(err, ErrArtifactWrongFieldFormat)
	}
	return bytecode, nil
}

func GetConstructorArgTypes(artifactJson string) ([]string, error) {

	value := gjson.Get(artifactJson, "abi.#(type==\"constructor\").inputs.#.type")
	if !value.Exists() {
		return nil, ErrArtifactFieldNotFound
	}

	valueArray := value.Array()

	args := make([]string, len(valueArray))
	for i := range valueArray {
		args[i] = valueArray[i].String()
	}

	return args, nil
}

func GetAbi(artifactJson string) (string, error) {
	value := gjson.Get(artifactJson, "abi")
	if !value.Exists() {
		return "", ErrArtifactFieldNotFound
	}

	return value.String(), nil
}
