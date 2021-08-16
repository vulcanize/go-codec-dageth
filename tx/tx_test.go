package tx_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
	"github.com/vulcanize/go-codec-dageth/tx"
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

	legacyTxConsensusEnc, alTxConsensusEnc, dfTxConsensusEnc []byte
	legacyTxNode, accessListTxNode, dynamicFeeTxNode         ipld.Node
)

/* IPLD Schemas
type StorageKeys [Hash]

type AccessElement struct {
	Address     Address
	StorageKeys StorageKeys
}

type AccessList [AccessElement]

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

func TestTransactionCodec(t *testing.T) {
	var err error
	legacyTxConsensusEnc, err = legacyTx.MarshalBinary()
	if err != nil {
		t.Fatalf("unable to marshal legacy transaction binary: %v", err)
	}
	alTxConsensusEnc, err = accessListTx.MarshalBinary()
	if err != nil {
		t.Fatalf("unable to marshal access list transaction binary: %v", err)
	}
	dfTxConsensusEnc, err = dynamicFeeTx.MarshalBinary()
	if err != nil {
		t.Fatalf("unable to marshal dynamic fee transaction binary: %v", err)
	}
	testTransactionDecoding(t)
	shared.TestAccessListTransactionNodeContent(t, accessListTxNode, accessListTx)
	shared.TestDynamicFeeTransactionNodeContent(t, dynamicFeeTxNode, dynamicFeeTx)
	shared.TestLegacyTransactionNodeContent(t, legacyTxNode, legacyTx)
	testTransactionEncoding(t)
}

func testTransactionDecoding(t *testing.T) {
	legacyTxBuilder := dageth.Type.Transaction.NewBuilder()
	legacyTxReader := bytes.NewReader(legacyTxConsensusEnc)
	if err := tx.Decode(legacyTxBuilder, legacyTxReader); err != nil {
		t.Fatalf("unable to decode legacy transaction into an IPLD node: %v", err)
	}
	legacyTxNode = legacyTxBuilder.Build()

	alTxBuilder := dageth.Type.Transaction.NewBuilder()
	alTxReader := bytes.NewReader(alTxConsensusEnc)
	if err := tx.Decode(alTxBuilder, alTxReader); err != nil {
		t.Fatalf("unable to decode access list transaction into an IPLD node: %v", err)
	}
	accessListTxNode = alTxBuilder.Build()

	dfTxBuilder := dageth.Type.Transaction.NewBuilder()
	dfTxReader := bytes.NewReader(dfTxConsensusEnc)
	if err := tx.Decode(dfTxBuilder, dfTxReader); err != nil {
		t.Fatalf("unable to decode dynamic fee transaction into an IPLD node: %v", err)
	}
	dynamicFeeTxNode = dfTxBuilder.Build()
}

func testTransactionEncoding(t *testing.T) {
	legTxWriter := new(bytes.Buffer)
	if err := tx.Encode(legacyTxNode, legTxWriter); err != nil {
		t.Fatalf("unable to encode legacy receipt into writer: %v", err)
	}
	legTxBytes := legTxWriter.Bytes()
	if !bytes.Equal(legTxBytes, legacyTxConsensusEnc) {
		t.Errorf("legacy transaction encoding (%x) does not match the expected consensus encoding (%x)", legTxBytes, legacyTxConsensusEnc)
	}

	alTxWriter := new(bytes.Buffer)
	if err := tx.Encode(accessListTxNode, alTxWriter); err != nil {
		t.Fatalf("unable to encode access list transaction into writer: %v", err)
	}
	alTxBytes := alTxWriter.Bytes()
	if !bytes.Equal(alTxBytes, alTxConsensusEnc) {
		t.Errorf("access list transaction encoding (%x) does not match the expected consensus encoding (%x)", alTxBytes, alTxConsensusEnc)
	}

	dfTxWriter := new(bytes.Buffer)
	if err := tx.Encode(dynamicFeeTxNode, dfTxWriter); err != nil {
		t.Fatalf("unable to encode access list transaction into writer: %v", err)
	}
	dfTxBytes := dfTxWriter.Bytes()
	if !bytes.Equal(dfTxBytes, dfTxConsensusEnc) {
		t.Errorf("dynamic fee transaction encoding (%x) does not match the expected consensus encoding (%x)", dfTxBytes, dfTxConsensusEnc)
	}
}
