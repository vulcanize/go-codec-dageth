package tx_test

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/tx"
)

var (
	testAddr        = common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b")
	testAddr2       = common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf1a")
	testStorageKey  = crypto.Keccak256Hash(testAddr.Bytes())
	testStorageKey2 = crypto.Keccak256Hash(testAddr2.Bytes())

	legacyTx, _ = types.NewTransaction(
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

	legacyTxConsensusEnc, alTxConsensusEnc []byte
	legacyTxNode, accessListTxNode         ipld.Node
)

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
	testTransactionDecoding(t)
	testAccessListTransactionNodeContent(t)
	testLegacyTransactionNodeContent(t)
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
}

/*
type Transaction struct {
			Type         TxType
			// We could make ChainID a required field in the IPLD schema
			ChainID      nullable BigInt # null unless the transaction is an EIP-2930 transaction
			AccountNonce Uint
			GasPrice     BigInt
			GasLimit     Uint
			Recipient    nullable Address # null recipient means the tx is a contract creation
			Amount       BigInt
			Data         Bytes
			AccessList   nullable AccessList # null unless the transaction is an EIP-2930 transaction

			# Signature values
			V            BigInt
			R            BigInt
			S            BigInt
		}

type StorageKeys [Hash]

		type AccessElement struct {
		    Address     Address
		    StorageKeys StorageKeys
		}
*/

func testAccessListTransactionNodeContent(t *testing.T) {
	verifySharedContent(t, accessListTxNode, accessListTx)
	accessListNode, err := accessListTxNode.LookupByString("AccessList")
	if err != nil {
		t.Fatalf("transaction missing AccessList")
	}
	if accessListNode.IsNull() {
		t.Fatalf("access list transaction AccessList should not be null")
	}
	if accessListNode.Length() != int64(len(accessListTx.AccessList())) {
		t.Fatalf("transaction access list should have %d elements", len(accessListTx.AccessList()))
	}
	accessListIT := accessListNode.ListIterator()
	for !accessListIT.Done() {
		i, accessListElementNode, err := accessListIT.Next()
		if err != nil {
			t.Fatalf("transaction access list iterator error: %v", err)
		}
		currentAccessListElement := accessListTx.AccessList()[i]
		addressNode, err := accessListElementNode.LookupByString("Address")
		if err != nil {
			t.Fatalf("transaction access list missing Address")
		}
		addressBytes, err := addressNode.AsBytes()
		if err != nil {
			t.Fatalf("transaction access list Address should be of type Bytes")
		}
		if !bytes.Equal(addressBytes, currentAccessListElement.Address.Bytes()) {
			t.Errorf("transaction access list address (%x) does not match expected address (%x)", addressBytes, currentAccessListElement.Address.Bytes())
		}

		storageKeysNode, err := accessListElementNode.LookupByString("StorageKeys")
		if err != nil {
			t.Fatalf("transaction access list missing StorageKeys")
		}
		if storageKeysNode.Length() != int64(len(currentAccessListElement.StorageKeys)) {
			t.Fatalf("transaction access list storage keys should have %d entries", len(currentAccessListElement.StorageKeys))
		}
		storageKeyIT := storageKeysNode.ListIterator()
		for !storageKeyIT.Done() {
			j, storageKeyNode, err := storageKeyIT.Next()
			if err != nil {
				t.Fatalf("transaction access list storage keys iterator error: %v", err)
			}
			currentStorageKey := currentAccessListElement.StorageKeys[j]
			storageKeyBytes, err := storageKeyNode.AsBytes()
			if err != nil {
				t.Fatalf("transaction access list StorageKey should be of type Bytes")
			}
			if !bytes.Equal(storageKeyBytes, currentStorageKey.Bytes()) {
				t.Errorf("transaction access list storage key (%x) does not match expected value (%x)", storageKeyBytes, currentStorageKey.Bytes())
			}
		}
	}

	idNode, err := accessListTxNode.LookupByString("ChainID")
	if err != nil {
		t.Fatalf("transaction is missing ChainID")
	}
	if idNode.IsNull() {
		t.Fatalf("access list transaction ChainID should not be null")
	}
	idBytes, err := idNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction ChainID should be of type Bytes")
	}
	if !bytes.Equal(idBytes, accessListTx.ChainId().Bytes()) {
		t.Errorf("transaction chain id (%x) does not match expected status (%x)", idBytes, accessListTx.ChainId().Bytes())
	}
}

func testLegacyTransactionNodeContent(t *testing.T) {
	verifySharedContent(t, legacyTxNode, legacyTx)
}

func verifySharedContent(t *testing.T, txNode ipld.Node, tx *types.Transaction) {
	v, r, s := tx.RawSignatureValues()
	vNode, err := txNode.LookupByString("V")
	if err != nil {
		t.Fatalf("transaction missing V")
	}
	vBytes, err := vNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction V should be of type Bytes")
	}
	if !bytes.Equal(vBytes, v.Bytes()) {
		t.Errorf("transaction v bytes (%x) does not match expected bytes (%x)", vBytes, v.Bytes())
	}

	rNode, err := txNode.LookupByString("R")
	if err != nil {
		t.Fatalf("transaction missing R")
	}
	rBytes, err := rNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction R should be of type Bytes")
	}
	if !bytes.Equal(rBytes, r.Bytes()) {
		t.Errorf("transaction r bytes (%x) does not match expected bytes (%x)", rBytes, r.Bytes())
	}

	sNode, err := txNode.LookupByString("S")
	if err != nil {
		t.Fatalf("transaction missing S")
	}
	sBytes, err := sNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction S should be of type Bytes")
	}
	if !bytes.Equal(sBytes, s.Bytes()) {
		t.Errorf("transaction s bytes (%x) does not match expected bytes (%x)", sBytes, s.Bytes())
	}

	dataNode, err := txNode.LookupByString("Data")
	if err != nil {
		t.Fatalf("transaction missing Data")
	}
	dataBytes, err := dataNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Data should be of type Bytes")
	}
	if !bytes.Equal(dataBytes, tx.Data()) {
		t.Errorf("transaction data bytes (%x) does not match expected bytes (%x)", dataBytes, tx.Data())
	}

	amountNode, err := txNode.LookupByString("Amount")
	if err != nil {
		t.Fatalf("transaction missing Amount")
	}
	amountBytes, err := amountNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Amount should be of type Bytes")
	}
	if !bytes.Equal(amountBytes, tx.Value().Bytes()) {
		t.Errorf("transaction amount (%x) does not match expected amount (%x)", amountBytes, tx.Value().Bytes())
	}

	recipientNode, err := txNode.LookupByString("Recipient")
	if err != nil {
		t.Fatalf("transaction missing Recipient")
	}
	recipientBytes, err := recipientNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Recipient should be of type Bytes")
	}
	if !bytes.Equal(recipientBytes, tx.To().Bytes()) {
		t.Errorf("transaction recipient (%x) does not match expected recipient (%x)", recipientBytes, tx.To().Bytes())
	}

	gasLimitNode, err := txNode.LookupByString("GasLimit")
	if err != nil {
		t.Fatalf("transaction missing GasLimit")
	}
	gasLimitBytes, err := gasLimitNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasLimit should be of type Bytes")
	}
	gas := binary.BigEndian.Uint64(gasLimitBytes)
	if gas != tx.Gas() {
		t.Errorf("transaction gas limit (%d) does not match expected gas limit (%d)", gas, tx.Gas())
	}

	gasPriceNode, err := txNode.LookupByString("GasPrice")
	if err != nil {
		t.Fatalf("transaction missing GasPrice")
	}
	gasPriceBytes, err := gasPriceNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasPrice should be of type Bytes")
	}
	if !bytes.Equal(gasPriceBytes, tx.GasPrice().Bytes()) {
		t.Errorf("transaction gas price (%x) does not match expected gas price (%x)", gasPriceBytes, tx.GasPrice().Bytes())
	}

	nonceNode, err := txNode.LookupByString("AccountNonce")
	if err != nil {
		t.Fatalf("transaction missing AccountNonce")
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Nonce should be of type Bytes")
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	if nonce != tx.Nonce() {
		t.Errorf("transaction nonce (%d) does not match expected nonce (%d)", nonce, tx.Nonce())
	}

	typeNode, err := txNode.LookupByString("TxType")
	if err != nil {
		t.Fatalf("transaction missing TxType")
	}
	typeBy, err := typeNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction TxType should be of type Bytes")
	}
	if len(typeBy) != 1 {
		t.Fatalf("transaction TxType should be a single byte")
	}
	if typeBy[0] != tx.Type() {
		t.Errorf("transaction tx type (%d) does not match expected tx type (%d)", typeBy[0], tx.Type())
	}
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
}
