package tx_list_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
	"github.com/vulcanize/go-codec-dageth/tx_list"
)

var (
	testAddr        = common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b")
	testAddr2       = common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf1a")
	testStorageKey  = crypto.Keccak256Hash(testAddr.Bytes())
	testStorageKey2 = crypto.Keccak256Hash(testAddr2.Bytes())
	legacyTx, _     = types.NewTransaction(
		3,
		testAddr,
		big.NewInt(10),
		2000,
		big.NewInt(1),
		common.FromHex("5544"),
	).WithSignature(
		types.HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)

	accessListTx, _ = types.NewTx(&types.AccessListTx{
		ChainID:  big.NewInt(1),
		Nonce:    3,
		To:       &testAddr,
		Value:    big.NewInt(10),
		Gas:      25000,
		GasPrice: big.NewInt(1),
		Data:     common.FromHex("5544"),
		AccessList: types.AccessList{
			types.AccessTuple{
				Address: testAddr,
				StorageKeys: []common.Hash{
					testStorageKey,
					testStorageKey2,
				},
			},
			types.AccessTuple{
				Address:     testAddr2,
				StorageKeys: nil,
			},
		},
	}).WithSignature(
		types.NewEIP2930Signer(big.NewInt(1)),
		common.Hex2Bytes("c9519f4f2b30335884581971573fadf60c6204f59a911df35ee8a540456b266032f1e8e2c5dd761f9e4f88f41c8310aeaba26a8bfcdacfedfa12ec3862d3752101"),
	)

	dynamicFeeTx, _ = types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     3,
		To:        &testAddr,
		Value:     big.NewInt(10),
		Gas:       25000,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		Data:      common.FromHex("5544"),
		AccessList: types.AccessList{
			types.AccessTuple{
				Address: testAddr,
				StorageKeys: []common.Hash{
					testStorageKey,
					testStorageKey2,
				},
			},
			types.AccessTuple{
				Address:     testAddr2,
				StorageKeys: nil,
			},
		},
	}).WithSignature(
		types.NewLondonSigner(big.NewInt(1)),
		common.Hex2Bytes("c9519f4f2b30335884581971573fadf60c6204f59a911df35ee8a540456b266032f1e8e2c5dd761f9e4f88f41c8310aeaba26a8bfcdacfedfa12ec3862d3752101"),
	)
	txs       = []*types.Transaction{legacyTx, accessListTx, dynamicFeeTx}
	txsRLP, _ = rlp.EncodeToBytes(txs)
	txsNode   ipld.Node
)

/* IPLD Schema
type Transactions [Transaction]

type Transaction struct {
	Type         TxType
	ChainID      nullable BigInt # null unless the transaction is an EIP-2930 or EIP-1559 transaction
	AccountNonce Uint
	GasPrice     nullable BigInt # null if the transaction is an EIP-1559 transaction
	GasTipCap    nullable BigInt # null unless the transaciton is an EIP-1559 transaction
	GasFeeCap    nullable BigInt # null unless the transaction is an EIP-1559 transaction
	GasLimit     Uint
	Recipient    nullable Address # null recipient means the tx is a contract creation tx
	Amount       BigInt
	Data         Bytes
	AccessList   nullable AccessList # null unless the transaction is an EIP-2930 or EIP-1559 transaction

	# Signature values
	V            BigInt
	R            BigInt
	S            BigInt
}
*/

func TestTransactionsCodec(t *testing.T) {
	testTransactionsDecode(t)
	testTransactionsNodeContents(t)
	testTransactionsEncode(t)
}

func testTransactionsDecode(t *testing.T) {
	txsBuilder := dageth.Type.Transactions.NewBuilder()
	txsReader := bytes.NewReader(txsRLP)
	if err := tx_list.Decode(txsBuilder, txsReader); err != nil {
		t.Fatalf("unable to decode transactions into an IPLD node: %v", err)
	}
	txsNode = txsBuilder.Build()
}

func testTransactionsNodeContents(t *testing.T) {
	txsIT := txsNode.ListIterator()
	if int(txsNode.Length()) != len(txs) {
		t.Fatalf("tx list should have %d txs, got %d", len(txs), txsNode.Length())
	}
	for !txsIT.Done() {
		i, txNode, err := txsIT.Next()
		if err != nil {
			t.Fatalf("tx list iterator error: %v", err)
		}
		switch i {
		case 0:
			shared.TestLegacyTransactionNodeContent(t, txNode, legacyTx)
		case 1:
			shared.TestAccessListTransactionNodeContent(t, txNode, accessListTx)
		case 2:
			shared.TestDynamicFeeTransactionNodeContent(t, txNode, dynamicFeeTx)
		}
	}
}

func testTransactionsEncode(t *testing.T) {
	txsWriter := new(bytes.Buffer)
	if err := tx_list.Encode(txsNode, txsWriter); err != nil {
		t.Fatalf("unable to encode tx list into writer: %v", err)
	}
	encodedTransactionsBytes := txsWriter.Bytes()
	if !bytes.Equal(encodedTransactionsBytes, txsRLP) {
		t.Errorf("transaction list encoding (%x) does not match the expected RLP encoding (%x)", encodedTransactionsBytes, txsRLP)
	}
}
