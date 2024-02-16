package provider

import (
	"context"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

type SimulatedClient struct {
	b *backends.SimulatedBackend
}

// CallContract implements EvmClient.
func (c SimulatedClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return c.b.CallContract(ctx, call, blockNumber)
}

// ChainID implements EvmClient.
func (SimulatedClient) ChainID(context.Context) (*big.Int, error) {
	return big.NewInt(1337), nil
}

// CodeAt implements EvmClient.
func (c SimulatedClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return c.b.CodeAt(ctx, contract, blockNumber)
}

// EstimateGas implements EvmClient.
func (c SimulatedClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	return c.b.EstimateGas(ctx, call)
}

// FilterLogs implements EvmClient.
func (c SimulatedClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return c.b.FilterLogs(ctx, query)
}

// HeaderByNumber implements EvmClient.
func (c SimulatedClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return c.b.HeaderByNumber(ctx, number)
}

// PendingCodeAt implements EvmClient.
func (c SimulatedClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return c.b.PendingCodeAt(ctx, account)
}

// PendingNonceAt implements EvmClient.
func (c SimulatedClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return c.b.PendingNonceAt(ctx, account)
}

// SendTransaction implements EvmClient.
func (c SimulatedClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if err := c.b.SendTransaction(ctx, tx); err != nil {
		return err
	}
	c.b.Commit()
	return nil
}

// SubscribeFilterLogs implements EvmClient.
func (c SimulatedClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return c.b.SubscribeFilterLogs(ctx, query, ch)
}

// SuggestGasPrice implements EvmClient.
func (c SimulatedClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.b.SuggestGasPrice(ctx)
}

// SuggestGasTipCap implements EvmClient.
func (c SimulatedClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return c.b.SuggestGasTipCap(ctx)
}

// TransactionReceipt implements EvmClient.
func (c SimulatedClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return c.b.TransactionReceipt(ctx, txHash)
}

var faucetAddr common.Address
var faucetPk string

func createSimulatedClient() EvmClient {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic("Cannot generate a random key")
	}
	faucetAddr = crypto.PubkeyToAddress(privateKey.PublicKey)
	privateKeyBytes := crypto.FromECDSA(privateKey)
	faucetPk = hexutil.Encode(privateKeyBytes)[2:]

	addr := map[common.Address]core.GenesisAccount{
		faucetAddr: {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
	}
	alloc := core.GenesisAlloc(addr)
	return SimulatedClient{backends.NewSimulatedBackend(alloc, 9000000)}
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"evm": providerserver.NewProtocol6WithError(New("", createSimulatedClient())()),
}
