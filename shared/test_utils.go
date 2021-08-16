package shared

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ipld/go-ipld-prime"
)

// RandomHash returns a random hash
func RandomHash() common.Hash {
	rand.Seed(time.Now().UnixNano())
	hash := make([]byte, 32)
	rand.Read(hash)
	return common.BytesToHash(hash)
}

// RandomAddr returns a random address
func RandomAddr() common.Address {
	rand.Seed(time.Now().UnixNano())
	addr := make([]byte, 20)
	rand.Read(addr)
	return common.BytesToAddress(addr)
}

// RandomBytes returns a random byte slice of the provided length
func RandomBytes(len int) []byte {
	rand.Seed(time.Now().UnixNano())
	by := make([]byte, len)
	rand.Read(by)
	return by
}

// TestDynamicFeeTransactionNodeContent checks the contents a dynamic fee tx IPLD node against a provided tx
func TestDynamicFeeTransactionNodeContent(t *testing.T, txNode ipld.Node, tx *types.Transaction) {
	verifySharedTxContent(t, txNode, tx)
	verifySharedTxContent2(t, txNode, tx)

	gasTipCapNode, err := txNode.LookupByString("GasTipCap")
	if err != nil {
		t.Fatalf("transaction missing GasTipCap: %v", err)
	}
	gasTipCapBytes, err := gasTipCapNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasTipCap should be of type Bytes: %v", err)
	}
	if !bytes.Equal(gasTipCapBytes, tx.GasTipCap().Bytes()) {
		t.Errorf("transaction gas tip cap (%x) does not match expected gas tip cap (%x)", gasTipCapBytes, tx.GasTipCap().Bytes())
	}

	gasFeeCapNode, err := txNode.LookupByString("GasFeeCap")
	if err != nil {
		t.Fatalf("transaction missing GasFeeCap: %v", err)
	}
	gasFeeCapBytes, err := gasFeeCapNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasFeeCap should be of type Bytes: %v", err)
	}
	if !bytes.Equal(gasFeeCapBytes, tx.GasFeeCap().Bytes()) {
		t.Errorf("transaction gas fee cap (%x) does not match expected gas fee cap (%x)", gasFeeCapBytes, tx.GasFeeCap().Bytes())
	}
}

// TestAccessListTransactionNodeContent checks the content of a access list tx IPLD node against a provided tx
func TestAccessListTransactionNodeContent(t *testing.T, txNode ipld.Node, tx *types.Transaction) {
	verifySharedTxContent(t, txNode, tx)
	verifySharedTxContent2(t, txNode, tx)
	gasPriceNode, err := txNode.LookupByString("GasPrice")
	if err != nil {
		t.Fatalf("transaction missing GasPrice: %v", err)
	}
	gasPriceBytes, err := gasPriceNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasPrice should be of type Bytes: %v", err)
	}
	if !bytes.Equal(gasPriceBytes, tx.GasPrice().Bytes()) {
		t.Errorf("transaction gas price (%x) does not match expected gas price (%x)", gasPriceBytes, tx.GasPrice().Bytes())
	}
}

// TestLegacyTransactionNodeContent checks the contents of a legacy tx IPLD node against a provided tx
func TestLegacyTransactionNodeContent(t *testing.T, txNode ipld.Node, tx *types.Transaction) {
	verifySharedTxContent(t, txNode, tx)
	gasPriceNode, err := txNode.LookupByString("GasPrice")
	if err != nil {
		t.Fatalf("transaction missing GasPrice: %v", err)
	}
	gasPriceBytes, err := gasPriceNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasPrice should be of type Bytes: %v", err)
	}
	if !bytes.Equal(gasPriceBytes, tx.GasPrice().Bytes()) {
		t.Errorf("transaction gas price (%x) does not match expected gas price (%x)", gasPriceBytes, tx.GasPrice().Bytes())
	}
}

// verifySharedTxContent verifies the content shared between all 3 tx types
func verifySharedTxContent(t *testing.T, txNode ipld.Node, tx *types.Transaction) {
	v, r, s := tx.RawSignatureValues()
	vNode, err := txNode.LookupByString("V")
	if err != nil {
		t.Fatalf("transaction missing V: %v", err)
	}
	vBytes, err := vNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction V should be of type Bytes: %v", err)
	}
	if !bytes.Equal(vBytes, v.Bytes()) {
		t.Errorf("transaction v bytes (%x) does not match expected bytes (%x)", vBytes, v.Bytes())
	}

	rNode, err := txNode.LookupByString("R")
	if err != nil {
		t.Fatalf("transaction missing R: %v", err)
	}
	rBytes, err := rNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction R should be of type Bytes: %v", err)
	}
	if !bytes.Equal(rBytes, r.Bytes()) {
		t.Errorf("transaction r bytes (%x) does not match expected bytes (%x)", rBytes, r.Bytes())
	}

	sNode, err := txNode.LookupByString("S")
	if err != nil {
		t.Fatalf("transaction missing S: %v", err)
	}
	sBytes, err := sNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction S should be of type Bytes: %v", err)
	}
	if !bytes.Equal(sBytes, s.Bytes()) {
		t.Errorf("transaction s bytes (%x) does not match expected bytes (%x)", sBytes, s.Bytes())
	}

	dataNode, err := txNode.LookupByString("Data")
	if err != nil {
		t.Fatalf("transaction missing Data: %v", err)
	}
	dataBytes, err := dataNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Data should be of type Bytes: %v", err)
	}
	if !bytes.Equal(dataBytes, tx.Data()) {
		t.Errorf("transaction data bytes (%x) does not match expected bytes (%x)", dataBytes, tx.Data())
	}

	amountNode, err := txNode.LookupByString("Amount")
	if err != nil {
		t.Fatalf("transaction missing Amount: %v", err)
	}
	amountBytes, err := amountNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Amount should be of type Bytes: %v", err)
	}
	if !bytes.Equal(amountBytes, tx.Value().Bytes()) {
		t.Errorf("transaction amount (%x) does not match expected amount (%x)", amountBytes, tx.Value().Bytes())
	}

	recipientNode, err := txNode.LookupByString("Recipient")
	if err != nil {
		t.Fatalf("transaction missing Recipient: %v", err)
	}
	recipientBytes, err := recipientNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Recipient should be of type Bytes: %v", err)
	}
	if !bytes.Equal(recipientBytes, tx.To().Bytes()) {
		t.Errorf("transaction recipient (%x) does not match expected recipient (%x)", recipientBytes, tx.To().Bytes())
	}

	gasLimitNode, err := txNode.LookupByString("GasLimit")
	if err != nil {
		t.Fatalf("transaction missing GasLimit: %v", err)
	}
	gasLimitBytes, err := gasLimitNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasLimit should be of type Bytes: %v", err)
	}
	gas := binary.BigEndian.Uint64(gasLimitBytes)
	if gas != tx.Gas() {
		t.Errorf("transaction gas limit (%d) does not match expected gas limit (%d)", gas, tx.Gas())
	}

	nonceNode, err := txNode.LookupByString("AccountNonce")
	if err != nil {
		t.Fatalf("transaction missing AccountNonce: %v", err)
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Nonce should be of type Bytes: %v", err)
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	if nonce != tx.Nonce() {
		t.Errorf("transaction nonce (%d) does not match expected nonce (%d)", nonce, tx.Nonce())
	}

	typeNode, err := txNode.LookupByString("TxType")
	if err != nil {
		t.Fatalf("transaction missing TxType: %v", err)
	}
	typeBy, err := typeNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction TxType should be of type Bytes: %v", err)
	}
	if len(typeBy) != 1 {
		t.Fatalf("transaction TxType should be a single byte")
	}
	if typeBy[0] != tx.Type() {
		t.Errorf("transaction tx type (%d) does not match expected tx type (%d)", typeBy[0], tx.Type())
	}
}

// verifySharedTxContent2 verifies the content shared between access list and dynamic fee txs
func verifySharedTxContent2(t *testing.T, txNode ipld.Node, tx *types.Transaction) {
	accessListNode, err := txNode.LookupByString("AccessList")
	if err != nil {
		t.Fatalf("transaction missing AccessList: %v", err)
	}
	if accessListNode.IsNull() {
		t.Fatalf("access list transaction AccessList should not be null")
	}
	if accessListNode.Length() != int64(len(tx.AccessList())) {
		t.Fatalf("transaction access list should have %d elements", len(tx.AccessList()))
	}
	accessListIT := accessListNode.ListIterator()
	for !accessListIT.Done() {
		i, accessListElementNode, err := accessListIT.Next()
		if err != nil {
			t.Fatalf("transaction access list iterator error: %v", err)
		}
		currentAccessListElement := tx.AccessList()[i]
		addressNode, err := accessListElementNode.LookupByString("Address")
		if err != nil {
			t.Fatalf("transaction access list missing Address: %v", err)
		}
		addressBytes, err := addressNode.AsBytes()
		if err != nil {
			t.Fatalf("transaction access list Address should be of type Bytes: %v", err)
		}
		if !bytes.Equal(addressBytes, currentAccessListElement.Address.Bytes()) {
			t.Errorf("transaction access list address (%x) does not match expected address (%x)", addressBytes, currentAccessListElement.Address.Bytes())
		}

		storageKeysNode, err := accessListElementNode.LookupByString("StorageKeys")
		if err != nil {
			t.Fatalf("transaction access list missing StorageKeys: %v", err)
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
				t.Fatalf("transaction access list StorageKey should be of type Bytes: %v", err)
			}
			if !bytes.Equal(storageKeyBytes, currentStorageKey.Bytes()) {
				t.Errorf("transaction access list storage key (%x) does not match expected value (%x)", storageKeyBytes, currentStorageKey.Bytes())
			}
		}
	}

	idNode, err := txNode.LookupByString("ChainID")
	if err != nil {
		t.Fatalf("transaction is missing ChainID: %v", err)
	}
	if idNode.IsNull() {
		t.Fatalf("access list transaction ChainID should not be null")
	}
	idBytes, err := idNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction ChainID should be of type Bytes: %v", err)
	}
	if !bytes.Equal(idBytes, tx.ChainId().Bytes()) {
		t.Errorf("transaction chain id (%x) does not match expected status (%x)", idBytes, tx.ChainId().Bytes())
	}
}

// TestAccessListReceiptNodeContents checks the contents of a access list rct IPLD node agaisnt a provided receipt
func TestAccessListReceiptNodeContents(t *testing.T, rctNode ipld.Node, rct *types.Receipt) {
	verifySharedRctContent(t, rctNode, rct)
	statusNode, err := rctNode.LookupByString("Status")
	if err != nil {
		t.Fatalf("receipt is missing Status: %v", err)
	}
	if !statusNode.IsNull() {
		t.Fatalf("receipt Status should be null")
	}

	postStateNode, err := rctNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState: %v", err)
	}
	if postStateNode.IsNull() {
		t.Errorf("receipt PostState should not be null")
	}
	postStateBy, err := postStateNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt PostState should be of type Bytes: %v", err)
	}
	if !bytes.Equal(postStateBy, rct.PostState) {
		t.Errorf("receipt post state (%d) does not match expected post state (%d)", postStateBy, rct.PostState)
	}
}

// TestDynamicFeeReceiptNodeContents checks the contents of a dynamic fee rct IPLD node against a provided receipt
func TestDynamicFeeReceiptNodeContents(t *testing.T, rctNode ipld.Node, rct *types.Receipt) {
	verifySharedRctContent(t, rctNode, rct)
	statusNode, err := rctNode.LookupByString("Status")
	if err != nil {
		t.Fatalf("receipt is missing Status: %v", err)
	}
	if statusNode.IsNull() {
		t.Fatalf("receipt Status should not be null")
	}
	statusBy, err := statusNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt Status should be of type Bytes: %v", err)
	}
	status := binary.BigEndian.Uint64(statusBy)
	if status != rct.Status {
		t.Errorf("receipt status (%d) does not match expected status (%d)", status, rct.Status)
	}

	postStateNode, err := rctNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState: %v", err)
	}
	if !postStateNode.IsNull() {
		t.Errorf("receipt PostState should be null")
	}
}

// TestLegacyReceiptNodeContents checks the contents of a legacy rct IPLD node against a provided receipt
func TestLegacyReceiptNodeContents(t *testing.T, rctNode ipld.Node, rct *types.Receipt) {
	verifySharedRctContent(t, rctNode, rct)
	statusNode, err := rctNode.LookupByString("Status")
	if err != nil {
		t.Fatalf("receipt is missing Status: %v", err)
	}
	if statusNode.IsNull() {
		t.Fatalf("receipt Status should not be null")
	}
	statusBy, err := statusNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt Status should be of type Bytes: %v", err)
	}
	status := binary.BigEndian.Uint64(statusBy)
	if status != rct.Status {
		t.Errorf("receipt status (%d) does not match expected status (%d)", status, rct.Status)
	}

	postStateNode, err := rctNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState: %v", err)
	}
	if !postStateNode.IsNull() {
		t.Errorf("receipt PostState should be null")
	}
}

// verifySharedRctContent verifies the content shared between all 3 receipt types
func verifySharedRctContent(t *testing.T, rctNode ipld.Node, rct *types.Receipt) {
	typeNode, err := rctNode.LookupByString("TxType")
	if err != nil {
		t.Fatalf("receipt is missing TxType: %v", err)
	}
	typeBy, err := typeNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt TxType should be of type Bytes: %v", err)
	}
	if len(typeBy) != 1 {
		t.Fatalf("receipt TxType should be a single byte")
	}
	if typeBy[0] != rct.Type {
		t.Errorf("receipt tx type (%d) does not match expected tx type (%d)", typeBy[0], rct.Type)
	}

	cguNode, err := rctNode.LookupByString("CumulativeGasUsed")
	if err != nil {
		t.Fatalf("receipt is missing CumulativeGasUsed: %v", err)
	}
	cguBy, err := cguNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt CumulativeGasUsed should be of type Bytes: %v", err)
	}
	cgu := binary.BigEndian.Uint64(cguBy)
	if cgu != rct.CumulativeGasUsed {
		t.Errorf("receipt cumulative gas used (%d) does not match expected cumulative gas used (%d)", cgu, rct.CumulativeGasUsed)
	}

	bloomNode, err := rctNode.LookupByString("Bloom")
	if err != nil {
		t.Fatalf("receipt is missing Bloom: %v", err)
	}
	bloomBy, err := bloomNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt Bloom should be of type Bytes: %v", err)
	}
	if !bytes.Equal(bloomBy, rct.Bloom.Bytes()) {
		t.Errorf("receipt bloom (%x) does not match expected bloom (%x)", bloomBy, rct.Bloom.Bytes())
	}

	logsNode, err := rctNode.LookupByString("Logs")
	if err != nil {
		t.Fatalf("receipt is missing Logs: %v", err)
	}
	if logsNode.Length() != int64(len(rct.Logs)) {
		t.Fatalf("receipt should have %d logs", len(rct.Logs))
	}
	logsLI := logsNode.ListIterator()
	for !logsLI.Done() {
		i, logNode, err := logsLI.Next()
		if err != nil {
			t.Fatalf("receipt log iterator error: %v", err)
		}
		currentLog := rct.Logs[i]
		addrNode, err := logNode.LookupByString("Address")
		if err != nil {
			t.Fatalf("receipt log is missing Address: %v", err)
		}
		addrBy, err := addrNode.AsBytes()
		if err != nil {
			t.Fatalf("receipt log Address should be of type Bytes: %v", err)
		}
		if !bytes.Equal(addrBy, currentLog.Address.Bytes()) {
			t.Errorf("receipt log address (%x) does not match expected address (%x)", addrBy, currentLog.Address.Bytes())
		}
		dataNode, err := logNode.LookupByString("Data")
		if err != nil {
			t.Fatalf("receipt log is missing Data: %v", err)
		}
		data, err := dataNode.AsBytes()
		if err != nil {
			t.Fatalf("receipt log Data should be of type Bytes: %v", err)
		}
		if !bytes.Equal(data, currentLog.Data) {
			t.Errorf("receipt log data (%x) does not match expected data (%x)", data, currentLog.Data)
		}
		topicsNode, err := logNode.LookupByString("Topics")
		if err != nil {
			t.Fatalf("receipt log is missing Topics: %v", err)
		}
		if topicsNode.Length() != 2 {
			t.Fatal("receipt log should have two topics")
		}
		topicsLI := topicsNode.ListIterator()
		for !topicsLI.Done() {
			j, topicNode, err := topicsLI.Next()
			if err != nil {
				t.Fatalf("receipt log topic iterator error: %v", err)
			}
			currentTopic := currentLog.Topics[j].Bytes()
			topicBy, err := topicNode.AsBytes()
			if err != nil {
				t.Fatalf("receipt log Topic should be of type Bytes: %v", err)
			}
			if !bytes.Equal(topicBy, currentTopic) {
				t.Errorf("receipt log topic%d (%x) does not match expected topic%d (%x)", j, topicBy, j, currentTopic)
			}
		}
	}
}
