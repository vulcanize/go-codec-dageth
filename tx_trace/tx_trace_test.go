package tx_trace_test

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
	"github.com/vulcanize/go-codec-dageth/tx_trace"
)

var (
	txHashes = []common.Hash{
		shared.RandomHash(),
		shared.RandomHash(),
		shared.RandomHash(),
		shared.RandomHash(),
		shared.RandomHash(),
	}
	frame1 = tx_trace.Frame{
		Op:     vm.CALL,
		From:   shared.RandomAddr(),
		To:     shared.RandomAddr(),
		Input:  shared.RandomBytes(128),
		Output: shared.RandomBytes(32),
		Gas:    1_100_000,
		Cost:   100_000,
		Value:  big.NewInt(133337),
	}
	frame2 = tx_trace.Frame{
		Op:     vm.ADD,
		From:   shared.RandomAddr(),
		To:     shared.RandomAddr(),
		Input:  shared.RandomBytes(64),
		Output: shared.RandomBytes(8),
		Gas:    100_000,
		Cost:   50_000,
		Value:  big.NewInt(13337),
	}
	frame3 = tx_trace.Frame{
		Op:     vm.CALLCODE,
		From:   shared.RandomAddr(),
		To:     shared.RandomAddr(),
		Input:  shared.RandomBytes(20),
		Output: shared.RandomBytes(20),
		Gas:    50_000,
		Cost:   40_000,
		Value:  big.NewInt(1337),
	}
	frames = []tx_trace.Frame{
		frame1,
		frame2,
		frame3,
	}
	mockTrace = tx_trace.TxTrace{
		TxHashes:  txHashes,
		StateRoot: shared.RandomHash(),
		Result:    []byte("this is a fake result for testing purposes"),
		Frames:    frames,
		Gas:       1_100_000,
		Failed:    false,
	}
	traceEnc  []byte
	traceNode ipld.Node
)

/*
# TxTrace contains the EVM context, input, and output for each OPCODE in a transaction that was applied to a specific state
type TxTrace struct {
   TxCIDs TxCIDList
   # CID link to the root node of the state trie that the above transaction set was applied on top of to produce this trace
   StateRootCID &StateTrieNode
   Result Bytes
   Frames FrameList
   Gas Uint
   Failed Bool
}

# TxCIDList
# List of CIDs linking to the transactions that were used to generate this trace by applying them onto the state referenced below
# If this trace was produced by the first transaction in a block then this list will contain only that one transaction
# and this trace was produced by applying it directly to the referenced state
# Otherwise, the trace is the output of the last transaction in the list applied to the state produced by
# sequentially applying the proceeding txs to the referenced state
type TxCIDList [&Transaction]

# Frame represents the EVM context, input, and output for a specific OPCODE during a transaction trace
type Frame struct {
	Op     OpCode
	From   Address
	To     Address
	Input  Bytes
	Output Bytes
	Gas    Uint
	Cost   Uint
	Value  BigInt
}

type FrameList [Frame]
*/

func TestTxTraceCodec(t *testing.T) {
	var err error
	traceEnc, err = rlp.EncodeToBytes(mockTrace)
	if err != nil {
		t.Fatalf("unable to rlp encode tx trace: %v", err)
	}
	testTxTraceDecoding(t)
	testTxTraceNodeContents(t)
	testTxTraceEncoding(t)
}

func testTxTraceDecoding(t *testing.T) {
	txTraceBuilder := dageth.Type.TxTrace.NewBuilder()
	txTraceReader := bytes.NewReader(traceEnc)
	if err := tx_trace.Decode(txTraceBuilder, txTraceReader); err != nil {
		t.Fatalf("unable to decode tx trace into an IPLD node: %v", err)
	}
	traceNode = txTraceBuilder.Build()
}

func testTxTraceNodeContents(t *testing.T) {
	txCIDsNode, err := traceNode.LookupByString("TxCIDs")
	if err != nil {
		t.Fatalf("tx trace missing TxCIDs: %v", err)
	}
	cidsLen := txCIDsNode.Length()
	if int(cidsLen) != len(txHashes) {
		t.Fatalf("tx trace TxCIDs length (%d) does not match expected length (%d)", cidsLen, len(txHashes))
	}
	txCIDsIT := txCIDsNode.ListIterator()
	for !txCIDsIT.Done() {
		i, txCIDNode, err := txCIDsIT.Next()
		if err != nil {
			t.Fatalf("tx trace TxCIDs iterator error: %v", err)
		}
		txLink, err := txCIDNode.AsLink()
		if err != nil {
			t.Fatalf("tx trace TxCID %d should be of type Bytes: %v", i, err)
		}
		txCIDLink, ok := txLink.(cidlink.Link)
		if !ok {
			t.Fatalf("tx trace TxCID %d could not be resolved to a CID link", i)
		}
		txMh := txCIDLink.Hash()
		decodedTxMh, err := multihash.Decode(txMh)
		if !bytes.Equal(decodedTxMh.Digest, txHashes[i].Bytes()) {
			t.Errorf("tx trace TxCID %d (%x) does not match expected TxCID %d (%x)", i, decodedTxMh.Digest, i, txHashes[i].Bytes())
		}
	}

	srCIDNode, err := traceNode.LookupByString("StateRootCID")
	if err != nil {
		t.Fatalf("tx trace missing StateRootCID: %v", err)
	}
	srLink, err := srCIDNode.AsLink()
	if err != nil {
		t.Fatalf("tx trace StateRootCID should be of type Link: %v", err)
	}
	srCIDLink, ok := srLink.(cidlink.Link)
	if !ok {
		t.Fatalf("tx trace StateRootCID could not be resolved to a CID link")
	}
	srMh := srCIDLink.Hash()
	decodedSrMh, err := multihash.Decode(srMh)
	if !bytes.Equal(decodedSrMh.Digest, mockTrace.StateRoot.Bytes()) {
		t.Errorf("tx trace StateRootCID (%x) does not match expected StateRootCID (%x)", decodedSrMh.Digest, mockTrace.StateRoot.Bytes())
	}

	resultNode, err := traceNode.LookupByString("Result")
	if err != nil {
		t.Fatalf("tx trace missing Result: %v", err)
	}
	result, err := resultNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace Result should be of type Bytes: %v", err)
	}
	if !bytes.Equal(result, mockTrace.Result) {
		t.Errorf("tx trace Result (%x) does not match expected Result (%x)", result, mockTrace.Result)
	}

	framesNode, err := traceNode.LookupByString("Frames")
	if err != nil {
		t.Fatalf("tx trace missing Frames: %v", err)
	}
	framesLen := framesNode.Length()
	if int(framesLen) != len(frames) {
		t.Fatalf("tx trace Frames length (%d) does not match expected length (%d)", framesLen, len(frames))
	}
	framesIT := framesNode.ListIterator()
	for !framesIT.Done() {
		i, frameNode, err := framesIT.Next()
		if err != nil {
			t.Fatalf("tx trace Frames iterator error: %v", err)
		}
		testFrameNodeContents(frameNode, frames[i], t)
	}

	gasNode, err := traceNode.LookupByString("Gas")
	if err != nil {
		t.Fatalf("tx trace missing Gas: %v", err)
	}
	gasBytes, err := gasNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace Gas should be of type Bytes: %v", err)
	}
	gas := binary.BigEndian.Uint64(gasBytes)
	if gas != mockTrace.Gas {
		t.Errorf("tx trace Gas (%d) does not match expected Gas (%d)", gas, mockTrace.Gas)
	}

	failedNode, err := traceNode.LookupByString("Failed")
	if err != nil {
		t.Fatalf("tx trace missing Failed: %v", err)
	}
	failed, err := failedNode.AsBool()
	if err != nil {
		t.Fatalf("tx trace Failed should be of type Bool: %v", err)
	}
	if failed != mockTrace.Failed {
		t.Errorf("tx trace Failed (%t) does not match expected Failed (%t)", failed, mockTrace.Failed)
	}
}

func testFrameNodeContents(frameNode ipld.Node, frame tx_trace.Frame, t *testing.T) {
	opNode, err := frameNode.LookupByString("Op")
	if err != nil {
		t.Fatalf("tx trace frame is missing Op: %v", err)
	}
	opBytes, err := opNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace frame Op should be of type Bytes: %v", err)
	}
	if len(opBytes) != 1 {
		t.Fatalf("tx trace frame Op should be a single byte")
	}
	if vm.OpCode(opBytes[0]) != frame.Op {
		t.Errorf("tx trace frame Op (%x) does not match expected Op (%x)", opBytes[0], frame.Op)
	}

	fromNode, err := frameNode.LookupByString("From")
	if err != nil {
		t.Fatalf("tx trace frame is missing From: %v", err)
	}
	fromBytes, err := fromNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace frame From should be of type Bytes: %v", err)
	}
	if !bytes.Equal(fromBytes, frame.From.Bytes()) {
		t.Errorf("tx trace frame From (%x) does not match expected From (%x)", fromBytes, frame.From.Bytes())
	}

	toNode, err := frameNode.LookupByString("To")
	if err != nil {
		t.Fatalf("tx trace frame is missing To: %v", err)
	}
	toBytes, err := toNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace frame To should be of type Bytes: %v", err)
	}
	if !bytes.Equal(toBytes, frame.To.Bytes()) {
		t.Errorf("tx trace frame To (%x) does not match expected To (%x)", toBytes, frame.To.Bytes())
	}

	inputNode, err := frameNode.LookupByString("Input")
	if err != nil {
		t.Fatalf("tx trace frame is missing Input: %v", err)
	}
	inputBytes, err := inputNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace frame Input should be of type Bytes: %v", err)
	}
	if !bytes.Equal(inputBytes, frame.Input) {
		t.Errorf("tx trace frame Input (%x) does not match expected Input (%x)", inputBytes, frame.Input)
	}

	outputNode, err := frameNode.LookupByString("Output")
	if err != nil {
		t.Fatalf("tx trace frame is missing Output: %v", err)
	}
	outputBytes, err := outputNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace frame Output should be of type Bytes: %v", err)
	}
	if !bytes.Equal(outputBytes, frame.Output) {
		t.Errorf("tx trace frame Output (%x) does not match expected Output (%x)", outputBytes, frame.Output)
	}

	gasNode, err := frameNode.LookupByString("Gas")
	if err != nil {
		t.Fatalf("tx trace frame is missing Gas: %v", err)
	}
	gasBytes, err := gasNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace frame Gas should be of type Bytes: %v", err)
	}
	gas := binary.BigEndian.Uint64(gasBytes)
	if gas != frame.Gas {
		t.Errorf("tx trace frame Gas (%d) does not match expected Gas (%d)", gas, frame.Gas)
	}

	costNode, err := frameNode.LookupByString("Cost")
	if err != nil {
		t.Fatalf("tx trace frame is missing Cost: %v", err)
	}
	costBytes, err := costNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace frame Cost should be of type Bytes: %v", err)
	}
	cost := binary.BigEndian.Uint64(costBytes)
	if cost != frame.Cost {
		t.Errorf("tx trace frame Cost (%d) does not match expected Cost (%d)", cost, frame.Cost)
	}

	valueNode, err := frameNode.LookupByString("Value")
	if err != nil {
		t.Fatalf("tx trace frame is missing Value: %v", err)
	}
	valueBytes, err := valueNode.AsBytes()
	if err != nil {
		t.Fatalf("tx trace frame Value should be of type Bytes: %v", err)
	}
	value := new(big.Int).SetBytes(valueBytes)
	if value.Cmp(frame.Value) != 0 {
		t.Errorf("tx trace frame Value (%s) does not match expected Value (%s)", value.String(), frame.Value.String())
	}
}

func testTxTraceEncoding(t *testing.T) {
	txTraceWriter := new(bytes.Buffer)
	if err := tx_trace.Encode(traceNode, txTraceWriter); err != nil {
		t.Fatalf("unable to encode legacy receipt into writer: %v", err)
	}
	traceBytes := txTraceWriter.Bytes()
	if !bytes.Equal(traceBytes, traceEnc) {
		t.Errorf("tx trace encoding (%x) does not match the expected rlp encoding (%x)", traceBytes, traceEnc)
	}
}
