package rct_list_test

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/rct_list"
	"github.com/vulcanize/go-codec-dageth/shared"
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
	receipts       = []*types.Receipt{legacyReceipt, accessListReceipt, dynamicFeeReceipt}
	receiptsRLP, _ = rlp.EncodeToBytes(receipts)
	receiptsNode   ipld.Node
)

/* IPLD Schema
type Receipt struct {
	TxType			  TxType
	// We could make Status an enum
	Status	          Uint   // nullable
	PostState		  Hash   // nullable
	CumulativeGasUsed Uint
	Bloom             Bloom
	Logs 			  Logs
	LogRootCID        &TrieNode
}

type Receipts [Receipt]
*/

func TestReceiptsCodec(t *testing.T) {
	testReceiptsDecode(t)
	testReceiptsNodeContents(t)
	testReceiptsEncode(t)
}

func testReceiptsDecode(t *testing.T) {
	rctsBuilder := dageth.Type.Receipts.NewBuilder()
	rctsReader := bytes.NewReader(receiptsRLP)
	if err := rct_list.Decode(rctsBuilder, rctsReader); err != nil {
		t.Fatalf("unable to decode receipts into an IPLD node: %v", err)
	}
	receiptsNode = rctsBuilder.Build()
}

func testReceiptsNodeContents(t *testing.T) {
	receiptsIT := receiptsNode.ListIterator()
	if int(receiptsNode.Length()) != len(receipts) {
		t.Fatalf("rct list should have %d rcts, got %d", len(receipts), receiptsNode.Length())
	}
	for !receiptsIT.Done() {
		i, rctNode, err := receiptsIT.Next()
		if err != nil {
			t.Fatalf("rct list iterator error: %v", err)
		}
		switch i {
		case 0:
			shared.TestLegacyReceiptNodeContents(t, rctNode, legacyReceipt)
		case 1:
			shared.TestAccessListReceiptNodeContents(t, rctNode, accessListReceipt)
		case 2:
			shared.TestDynamicFeeReceiptNodeContents(t, rctNode, dynamicFeeReceipt)
		}
	}
}

func testReceiptsEncode(t *testing.T) {
	rctsWriter := new(bytes.Buffer)
	if err := rct_list.Encode(receiptsNode, rctsWriter); err != nil {
		t.Fatalf("unable to encode rct list into writer: %v", err)
	}
	encodedReceiptsBytes := rctsWriter.Bytes()
	if !bytes.Equal(encodedReceiptsBytes, receiptsRLP) {
		t.Errorf("receipt list encoding (%x) does not match the expected RLP encoding (%x)", encodedReceiptsBytes, receiptsRLP)
	}
}
