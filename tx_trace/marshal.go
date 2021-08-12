package tx_trace

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
)

// Encode provides an IPLD codec encode interface for eth transaction trace IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x9b (proposed) when this package is invoked via init.
func Encode(node ipld.Node, w io.Writer) error {
	// 1KiB can be allocated on the stack, and covers most small nodes
	// without having to grow the buffer and cause allocations.
	enc := make([]byte, 0, 1024)

	enc, err := AppendEncode(enc, node)
	if err != nil {
		return err
	}
	_, err = w.Write(enc)
	return err
}

// AppendEncode is like Encode, but it uses a destination buffer directly.
// This means less copying of bytes, and if the destination has enough capacity,
// fewer allocations.
func AppendEncode(enc []byte, inNode ipld.Node) ([]byte, error) {
	txTrace := new(TxTrace)
	if err := EncodeTxTrace(txTrace, inNode); err != nil {
		return nil, err
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	if err := rlp.Encode(wbs, txTrace); err != nil {
		return nil, err
	}
	return enc, nil
}

// EncodeTxTrace packs the node into a go-ethereum TxTrace
func EncodeTxTrace(txTrace *TxTrace, inNode ipld.Node) error {
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.TxTrace.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return err
	}
	node := builder.Build()
	for _, pFunc := range requiredPackFuncs {
		if err := pFunc(txTrace, node); err != nil {
			return err
		}
	}
	return nil
}

var requiredPackFuncs = []func(*TxTrace, ipld.Node) error{
	packTxCIDs,
	packStateRootCID,
	packResult,
	packFrames,
	packGas,
	packFailed,
}

func packTxCIDs(txTrace *TxTrace, node ipld.Node) error {
	txCIDList, err := node.LookupByString("TxCIDs")
	if err != nil {
		return err
	}
	txCIDListIT := txCIDList.ListIterator()
	txHashes := make([]common.Hash, txCIDList.Length())
	for !txCIDListIT.Done() {
		i, txCIDNode, err := txCIDListIT.Next()
		if err != nil {
			return err
		}
		txLink, err := txCIDNode.AsLink()
		if err != nil {
			return err
		}
		txCIDLink, ok := txLink.(cidlink.Link)
		if !ok {
			return fmt.Errorf("tx trace must have TxCIDs")
		}
		txMh := txCIDLink.Hash()
		decodedTxMh, err := multihash.Decode(txMh)
		if err != nil {
			return fmt.Errorf("unable to decode TxCID multihash: %v", err)
		}
		txHashes[i] = common.BytesToHash(decodedTxMh.Digest)
	}
	txTrace.TxHashes = txHashes
	return nil
}

func packStateRootCID(txTrace *TxTrace, node ipld.Node) error {
	srNode, err := node.LookupByString("StateRootCID")
	if err != nil {
		return err
	}
	srLink, err := srNode.AsLink()
	if err != nil {
		return err
	}
	srCIDLink, ok := srLink.(cidlink.Link)
	if !ok {
		return fmt.Errorf("tx trace must have a StateRootCID")
	}
	srMh := srCIDLink.Hash()
	decodedSrMh, err := multihash.Decode(srMh)
	if err != nil {
		return fmt.Errorf("unable to decode StateRootCID multihash: %v", err)
	}
	txTrace.StateRoot = common.BytesToHash(decodedSrMh.Digest)
	return nil
}

func packResult(txTrace *TxTrace, node ipld.Node) error {
	resNode, err := node.LookupByString("Result")
	if err != nil {
		return err
	}
	result, err := resNode.AsBytes()
	if err != nil {
		return err
	}
	txTrace.Result = result
	return nil
}

func packFrames(txTrace *TxTrace, node ipld.Node) error {
	frameList, err := node.LookupByString("Frames")
	if err != nil {
		return err
	}
	frameListIT := frameList.ListIterator()
	frames := make([]Frame, frameList.Length())
	for !frameListIT.Done() {
		i, frameNode, err := frameListIT.Next()
		if err != nil {
			return err
		}
		frame := new(Frame)
		if err := packFrame(frame, frameNode); err != nil {
			return err
		}
		frames[i] = *frame
	}
	txTrace.Frames = frames
	return nil
}

func packGas(txTrace *TxTrace, node ipld.Node) error {
	gasNode, err := node.LookupByString("Gas")
	if err != nil {
		return err
	}
	gasBytes, err := gasNode.AsBytes()
	if err != nil {
		return err
	}
	gas := binary.BigEndian.Uint64(gasBytes)
	txTrace.Gas = gas
	return nil
}

func packFailed(txTrace *TxTrace, node ipld.Node) error {
	failedNode, err := node.LookupByString("Failed")
	if err != nil {
		return err
	}
	failed, err := failedNode.AsBool()
	if err != nil {
		return err
	}
	txTrace.Failed = failed
	return nil
}

func packFrame(frame *Frame, node ipld.Node) error {
	for _, pFunc := range requiredFramePackFuncs {
		if err := pFunc(frame, node); err != nil {
			return err
		}
	}
	return nil
}

var requiredFramePackFuncs = []func(*Frame, ipld.Node) error{
	packOp,
	packFrom,
	packTo,
	packInput,
	packOutput,
	packFrameGas,
	packCost,
	packValue,
}

func packOp(frame *Frame, node ipld.Node) error {
	opNode, err := node.LookupByString("Op")
	if err != nil {
		return err
	}
	op, err := opNode.AsBytes()
	if err != nil {
		return err
	}
	frame.Op = vm.OpCode(op[0])
	return nil
}

func packFrom(frame *Frame, node ipld.Node) error {
	fromNode, err := node.LookupByString("From")
	if err != nil {
		return err
	}
	fromBytes, err := fromNode.AsBytes()
	if err != nil {
		return err
	}
	frame.From = common.BytesToAddress(fromBytes)
	return nil
}

func packTo(frame *Frame, node ipld.Node) error {
	toNode, err := node.LookupByString("To")
	if err != nil {
		return err
	}
	toBytes, err := toNode.AsBytes()
	if err != nil {
		return err
	}
	frame.To = common.BytesToAddress(toBytes)
	return nil
}

func packInput(frame *Frame, node ipld.Node) error {
	inNode, err := node.LookupByString("Input")
	if err != nil {
		return err
	}
	input, err := inNode.AsBytes()
	if err != nil {
		return err
	}
	frame.Input = input
	return nil
}

func packOutput(frame *Frame, node ipld.Node) error {
	outNode, err := node.LookupByString("Output")
	if err != nil {
		return err
	}
	output, err := outNode.AsBytes()
	if err != nil {
		return err
	}
	frame.Output = output
	return nil
}

func packFrameGas(frame *Frame, node ipld.Node) error {
	gasNode, err := node.LookupByString("Gas")
	if err != nil {
		return err
	}
	gasBytes, err := gasNode.AsBytes()
	if err != nil {
		return err
	}
	gas := binary.BigEndian.Uint64(gasBytes)
	frame.Gas = gas
	return nil
}

func packCost(frame *Frame, node ipld.Node) error {
	costNode, err := node.LookupByString("Cost")
	if err != nil {
		return err
	}
	costBytes, err := costNode.AsBytes()
	if err != nil {
		return err
	}
	cost := binary.BigEndian.Uint64(costBytes)
	frame.Cost = cost
	return nil
}

func packValue(frame *Frame, node ipld.Node) error {
	valNode, err := node.LookupByString("Value")
	if err != nil {
		return err
	}
	valBytes, err := valNode.AsBytes()
	if err != nil {
		return err
	}
	frame.Value = new(big.Int).SetBytes(valBytes)
	return nil
}
