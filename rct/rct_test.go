package rct_test

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/rct"
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
	dfReceiptConsensusEnc, err = dynamicFeeReceipt.MarshalBinary()
	if err != nil {
		t.Fatalf("unable to marshal dynamic fee receipt binary: %v", err)
	}
	testReceiptDecoding(t)
	shared.TestAccessListReceiptNodeContents(t, accessListReceiptNode, accessListReceipt)
	shared.TestDynamicFeeReceiptNodeContents(t, dynamicFeeReceiptNode, dynamicFeeReceipt)
	shared.TestLegacyReceiptNodeContents(t, legacyReceiptNode, legacyReceipt)
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

	dfRctBuilder := dageth.Type.Receipt.NewBuilder()
	dfRctReader := bytes.NewReader(dfReceiptConsensusEnc)
	if err := rct.Decode(dfRctBuilder, dfRctReader); err != nil {
		t.Fatalf("unable to decode dynamic fee receipt into an IPLD node: %v", err)
	}
	dynamicFeeReceiptNode = dfRctBuilder.Build()
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
