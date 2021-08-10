package rct_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/rct"
)

var (
	mockHash      = crypto.Keccak256([]byte{1, 2, 3, 4, 5})
	legacyReceipt = &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("hello"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("goodbye"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
		Type: types.LegacyTxType,
	}
	accessListReceipt = &types.Receipt{
		PostState:         mockHash,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("hello"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("goodbye"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
		Type: types.AccessListTxType,
	}
	dynamicFeeReceipt = &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("hello"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("goodbye"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
		Type: types.DynamicFeeTxType,
	}

	lReceiptConsensusEnc, alReceiptConsensusEnc, dfReceiptConsensusEnc []byte
	legacyReceiptNode, accessListReceiptNode, dynamicFeeReceiptNode    ipld.Node
)

/* IPLD Schemas
type Topics [Hash]

type Log struct {
	Address Address
	Topics  Topics
	Data    Bytes
}

type Logs [Log]

type Receipt struct {
	TxType			  TxType
	Status	          nullable Uint
	PostState		  nullable Hash
	CumulativeGasUsed Uint
	Bloom             Bloom
	Logs              Logs
	LogRootCID        &TrieNode
}

type Receipts [Receipt]
*/

func TestReceiptCodec(t *testing.T) {
	var err error
	lReceiptConsensusEnc, err = legacyReceipt.MarshalBinary()
	if err != nil {
		t.Fatalf("unable to marshal legacy receipt binary: %v", err)
	}
	alReceiptConsensusEnc, err = accessListReceipt.MarshalBinary()
	if err != nil {
		t.Fatalf("unable to marshal access list receipt binary: %v", err)
	}
	testReceiptDecoding(t)
	testAccessListReceiptNodeContents(t)
	testDynamicFeeReceiptNodeContents(t)
	testLegacyReceiptNodeContents(t)
	testReceiptEncoding(t)
}

func testReceiptDecoding(t *testing.T) {
	legacyRctBuilder := dageth.Type.Receipt.NewBuilder()
	legacyRctReader := bytes.NewReader(lReceiptConsensusEnc)
	if err := rct.Decode(legacyRctBuilder, legacyRctReader); err != nil {
		t.Fatalf("unable to decode legacy receipt into an IPLD node: %v", err)
	}
	legacyReceiptNode = legacyRctBuilder.Build()

	alRctBuilder := dageth.Type.Receipt.NewBuilder()
	alRctReader := bytes.NewReader(alReceiptConsensusEnc)
	if err := rct.Decode(alRctBuilder, alRctReader); err != nil {
		t.Fatalf("unable to decode access list receipt into an IPLD node: %v", err)
	}
	accessListReceiptNode = alRctBuilder.Build()
}

func testAccessListReceiptNodeContents(t *testing.T) {
	verifySharedContent(t, accessListReceiptNode, accessListReceipt)
	statusNode, err := accessListReceiptNode.LookupByString("Status")
	if err != nil {
		t.Fatalf("receipt is missing Status: %v", err)
	}
	if !statusNode.IsNull() {
		t.Fatalf("receipt Status should be null")
	}

	postStateNode, err := accessListReceiptNode.LookupByString("PostState")
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
	if !bytes.Equal(postStateBy, accessListReceipt.PostState) {
		t.Errorf("receipt post state (%d) does not match expected post state (%d)", postStateBy, accessListReceipt.PostState)
	}
}

func testDynamicFeeReceiptNodeContents(t *testing.T) {
	verifySharedContent(t, dynamicFeeReceiptNode, dynamicFeeReceipt)
	statusNode, err := dynamicFeeReceiptNode.LookupByString("Status")
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
	if status != dynamicFeeReceipt.Status {
		t.Errorf("receipt status (%d) does not match expected status (%d)", status, dynamicFeeReceipt.Status)
	}

	postStateNode, err := dynamicFeeReceiptNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState: %v", err)
	}
	if !postStateNode.IsNull() {
		t.Errorf("receipt PostState should be null")
	}
}

func testLegacyReceiptNodeContents(t *testing.T) {
	verifySharedContent(t, legacyReceiptNode, legacyReceipt)
	statusNode, err := legacyReceiptNode.LookupByString("Status")
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
	if status != legacyReceipt.Status {
		t.Errorf("receipt status (%d) does not match expected status (%d)", status, legacyReceipt.Status)
	}

	postStateNode, err := legacyReceiptNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState: %v", err)
	}
	if !postStateNode.IsNull() {
		t.Errorf("receipt PostState should be null")
	}
}

func verifySharedContent(t *testing.T, rctNode ipld.Node, rct *types.Receipt) {
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

func testReceiptEncoding(t *testing.T) {
	legRctWriter := new(bytes.Buffer)
	if err := rct.Encode(legacyReceiptNode, legRctWriter); err != nil {
		t.Fatalf("unable to encode legacy receipt into writer: %v", err)
	}
	legRctBytes := legRctWriter.Bytes()
	if !bytes.Equal(legRctBytes, lReceiptConsensusEnc) {
		t.Errorf("legacy receipt encoding (%x) does not match the expected consensus encoding (%x)", legRctBytes, lReceiptConsensusEnc)
	}

	alRctWriter := new(bytes.Buffer)
	if err := rct.Encode(accessListReceiptNode, alRctWriter); err != nil {
		t.Fatalf("unable to encode access list receipt into writer: %v", err)
	}
	alRctBytes := alRctWriter.Bytes()
	if !bytes.Equal(alRctBytes, alReceiptConsensusEnc) {
		t.Errorf("access list receipt encoding (%x) does not match the expected consensus encoding (%x)", alRctBytes, alReceiptConsensusEnc)
	}

	dfRctWriter := new(bytes.Buffer)
	if err := rct.Encode(dynamicFeeReceiptNode, dfRctWriter); err != nil {
		t.Fatalf("unable to encode access list receipt into writer: %v", err)
	}
	dfRctBytes := dfRctWriter.Bytes()
	if !bytes.Equal(dfRctBytes, dfReceiptConsensusEnc) {
		t.Errorf("dynamic fee receipt encoding (%x) does not match the expected consensus encoding (%x)", dfRctBytes, dfReceiptConsensusEnc)
	}
}
