package utils

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestGetBytecode(t *testing.T) {

	// Invalid JSON
	_, err := GetBytecode("{\"some\":\"stuff\"}")
	if !errors.Is(err, ErrArtifactFieldNotFound) {
		t.Fatalf("Expected error %v", err)
	}

	// Invalid field data format
	_, err = GetBytecode("{\"bytecode\":\"stuff\"}")
	if !errors.Is(err, ErrArtifactWrongFieldFormat) {
		t.Fatalf("Expected error %v", err)
	}
	_, err = GetBytecode("{\"bytecode\":\"00ffaa\"}")
	if !errors.Is(err, ErrArtifactWrongFieldFormat) {
		t.Fatalf("Expected error %v", err)
	}

	// Success
	b, err := os.ReadFile("../provider/testdata/Token.json")
	if err != nil {
		t.Fatalf("Cannot read test data file: %v", err)
		fmt.Print(err)
	}
	bytecode, err := GetBytecode(string(b))
	if err != nil {
		t.Fatalf("Unexpected error when reading bytecode: %v", err)
		fmt.Print(err)
	}
	expected := []byte{0x60, 0x80, 0x60, 0x40, 0x52, 0x34}
	if !reflect.DeepEqual(bytecode[:6], expected) {
		t.Fatalf("Wrong bytecode, expected %v, got %v", expected, bytecode)
		fmt.Print(err)
	}
}

func TestGetConstrutorArgs(t *testing.T) {
	// Success
	b, err := os.ReadFile("../provider/testdata/Token.json")
	if err != nil {
		t.Fatalf("Cannot read test data file: %v", err)
		fmt.Print(err)
	}
	args, err := GetConstructorArgTypes(string(b))
	if err != nil {
		t.Fatalf("Unexpected error when reading constructor args: %v", err)
		fmt.Print(err)
	}
	expected := []string{"string", "string", "uint256", "uint8"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Wrong bytecode, expected %v, got %v", expected, args)
		fmt.Print(err)
	}
}
