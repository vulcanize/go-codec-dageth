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

	lReceiptConsensusEnc, alReceiptConsensusEnc []byte
	legacyReceiptNode, accessListReceiptNode    ipld.Node
)

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
	alTypeNode, err := accessListReceiptNode.LookupByString("TxType")
	if err != nil {
		t.Fatalf("receipt is missing TxType")
	}
	alTypeBy, err := alTypeNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt TxType should be of type Bytes")
	}
	if len(alTypeBy) != 1 {
		t.Fatalf("receipt TxType should be a single byte")
	}
	if alTypeBy[0] != accessListReceipt.Type {
		t.Errorf("receipt tx type (%d) does not match expected tx type (%d)", alTypeBy[0], accessListReceipt.Type)
	}

	statusNode, err := accessListReceiptNode.LookupByString("Status")
	if err != nil {
		t.Fatalf("receipt is missing Status")
	}
	if !statusNode.IsNull() {
		t.Fatalf("receipt Status should be null")
	}

	postStateNode, err := accessListReceiptNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState")
	}
	if postStateNode.IsNull() {
		t.Errorf("receipt PostState should not be null")
	}
	postStateBy, err := postStateNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt PostState should be of type Bytes")
	}
	if !bytes.Equal(postStateBy, accessListReceipt.PostState) {
		t.Errorf("receipt post state (%d) does not match expected post state (%d)", postStateBy, accessListReceipt.PostState)
	}

	cguNode, err := accessListReceiptNode.LookupByString("CumulativeGasUsed")
	if err != nil {
		t.Fatalf("receipt is missing CumulativeGasUsed")
	}
	cguBy, err := cguNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt CumulativeGasUsed should be of type Bytes")
	}
	cgu := binary.BigEndian.Uint64(cguBy)
	if cgu != accessListReceipt.CumulativeGasUsed {
		t.Errorf("receipt cumulative gas used (%d) does not match expected cumulative gas used (%d)", cgu, accessListReceipt.CumulativeGasUsed)
	}

	bloomNode, err := accessListReceiptNode.LookupByString("Bloom")
	if err != nil {
		t.Fatalf("receipt is missing Bloom")
	}
	bloomBy, err := bloomNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt Bloom should be of type Bytes")
	}
	if !bytes.Equal(bloomBy, accessListReceipt.Bloom.Bytes()) {
		t.Errorf("receipt bloom (%x) does not match expected bloom (%x)", bloomBy, accessListReceipt.Bloom.Bytes())
	}

	logsNode, err := accessListReceiptNode.LookupByString("Logs")
	if err != nil {
		t.Fatalf("receipt is missing Logs")
	}
	if logsNode.Length() != 2 {
		t.Fatal("receipt should have two logs")
	}
	logsLI := logsNode.ListIterator()
	for !logsLI.Done() {
		i, logNode, err := logsLI.Next()
		if err != nil {
			t.Fatalf("receipt log iterator error: %v", err)
		}
		currentLog := accessListReceipt.Logs[i]
		addrNode, err := logNode.LookupByString("Address")
		if err != nil {
			t.Fatalf("receipt log is missing Address")
		}
		addrBy, err := addrNode.AsBytes()
		if err != nil {
			t.Fatalf("receipt log Address should be of type Bytes")
		}
		if !bytes.Equal(addrBy, currentLog.Address.Bytes()) {
			t.Errorf("receipt log address (%x) does not match expected address (%x)", addrBy, currentLog.Address.Bytes())
		}
		dataNode, err := logNode.LookupByString("Data")
		if err != nil {
			t.Fatalf("receipt log is missing Data")
		}
		data, err := dataNode.AsBytes()
		if err != nil {
			t.Fatalf("receipt log Data should be of type Bytes")
		}
		if !bytes.Equal(data, currentLog.Data) {
			t.Errorf("receipt log data (%x) does not match expected data (%x)", data, currentLog.Data)
		}
		topicsNode, err := logNode.LookupByString("Topics")
		if err != nil {
			t.Fatalf("receipt log is missing Topics")
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
				t.Fatalf("receipt log Topic should be of type Bytes")
			}
			if !bytes.Equal(topicBy, currentTopic) {
				t.Errorf("receipt log topic%d bytes (%x) does not match expected bytes (%x)", j, topicBy, currentTopic)
			}
		}
	}
}

func testLegacyReceiptNodeContents(t *testing.T) {
	typeNode, err := legacyReceiptNode.LookupByString("TxType")
	if err != nil {
		t.Fatalf("receipt is missing TxType")
	}
	typeBy, err := typeNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt TxType should be of type Bytes")
	}
	if len(typeBy) != 1 {
		t.Fatalf("receipt TxType should be a single byte")
	}
	if typeBy[0] != legacyReceipt.Type {
		t.Errorf("receipt tx type (%d) does not match expected tx type (%d)", typeBy[0], legacyReceipt.Type)
	}

	statusNode, err := legacyReceiptNode.LookupByString("Status")
	if err != nil {
		t.Fatalf("receipt is missing Status")
	}
	if statusNode.IsNull() {
		t.Fatalf("receipt Status should not be null")
	}
	statusBy, err := statusNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt Status should be of type Bytes")
	}
	status := binary.BigEndian.Uint64(statusBy)
	if status != legacyReceipt.Status {
		t.Errorf("receipt status (%d) does not match expected status (%d)", status, legacyReceipt.Status)
	}

	postStateNode, err := legacyReceiptNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState")
	}
	if !postStateNode.IsNull() {
		t.Errorf("receipt PostState should be null")
	}

	cguNode, err := legacyReceiptNode.LookupByString("CumulativeGasUsed")
	if err != nil {
		t.Fatalf("receipt is missing CumulativeGasUsed")
	}
	cguBy, err := cguNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt CumulativeGasUsed should be of type Bytes")
	}
	cgu := binary.BigEndian.Uint64(cguBy)
	if cgu != legacyReceipt.CumulativeGasUsed {
		t.Errorf("receipt cumulative gas used (%d) does not match expected cumulative gas used (%d)", cgu, legacyReceipt.CumulativeGasUsed)
	}

	bloomNode, err := legacyReceiptNode.LookupByString("Bloom")
	if err != nil {
		t.Fatalf("receipt is missing Bloom")
	}
	bloomBy, err := bloomNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt Bloom should be of type Bytes")
	}
	if !bytes.Equal(bloomBy, legacyReceipt.Bloom.Bytes()) {
		t.Errorf("receipt bloom (%x) does not match expected bloom (%x)", bloomBy, legacyReceipt.Bloom.Bytes())
	}

	logsNode, err := legacyReceiptNode.LookupByString("Logs")
	if err != nil {
		t.Fatalf("receipt is missing Logs")
	}
	if logsNode.Length() != 2 {
		t.Fatal("receipt should have two logs")
	}
	logsLI := logsNode.ListIterator()
	for !logsLI.Done() {
		i, logNode, err := logsLI.Next()
		if err != nil {
			t.Fatalf("receipt log iterator error: %v", err)
		}
		currentLog := legacyReceipt.Logs[i]
		addrNode, err := logNode.LookupByString("Address")
		if err != nil {
			t.Fatalf("receipt log is missing Address")
		}
		addrBy, err := addrNode.AsBytes()
		if err != nil {
			t.Fatalf("receipt log Address should be of type Bytes")
		}
		if !bytes.Equal(addrBy, currentLog.Address.Bytes()) {
			t.Errorf("receipt log address (%x) does not match expected address (%x)", addrBy, currentLog.Address.Bytes())
		}
		dataNode, err := logNode.LookupByString("Data")
		if err != nil {
			t.Fatalf("receipt log is missing Data")
		}
		data, err := dataNode.AsBytes()
		if err != nil {
			t.Fatalf("receipt log Data should be of type Bytes")
		}
		if !bytes.Equal(data, currentLog.Data) {
			t.Errorf("receipt log data (%x) does not match expected data (%x)", data, currentLog.Data)
		}
		topicsNode, err := logNode.LookupByString("Topics")
		if err != nil {
			t.Fatalf("receipt log is missing Topics")
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
				t.Fatalf("receipt log Topic should be of type Bytes")
			}
			if !bytes.Equal(topicBy, currentTopic) {
				t.Errorf("receipt log topic%d bytes (%x) does not match expected bytes (%x)", j, topicBy, currentTopic)
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
}
