package tx_trace

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	"github.com/vulcanize/go-codec-dageth/state_trie"
	"github.com/vulcanize/go-codec-dageth/tx"
)

// Decode provides an IPLD codec decode interface for eth transaction trace IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x9b (proposed) when this package is invoked via init.
func Decode(na ipld.NodeAssembler, in io.Reader) error {
	var src []byte
	if buf, ok := in.(interface{ Bytes() []byte }); ok {
		src = buf.Bytes()
	} else {
		var err error
		src, err = ioutil.ReadAll(in)
		if err != nil {
			return err
		}
	}
	return DecodeBytes(na, src)
}

// DecodeBytes is like Decode, but it uses an input buffer directly.
// Decode will grab or read all the bytes from an io.Reader anyway, so this can
// save having to copy the bytes or create a bytes.Buffer.
func DecodeBytes(na ipld.NodeAssembler, src []byte) error {
	var txTrace TxTrace
	if err := rlp.DecodeBytes(src, &txTrace); err != nil {
		return err
	}
	return DecodeTx(na, txTrace)
}

// DecodeTx unpacks a go-ethereum TxTrace into a NodeAssembler
func DecodeTx(na ipld.NodeAssembler, txTrace TxTrace) error {
	ma, err := na.BeginMap(14)
	if err != nil {
		return err
	}
	for _, upFunc := range requiredUnpackFuncs {
		if err := upFunc(ma, txTrace); err != nil {
			return fmt.Errorf("invalid DAG-ETH TxTrace binary (%v)", err)
		}
	}
	return ma.Finish()
}

var requiredUnpackFuncs = []func(ipld.MapAssembler, TxTrace) error{
	unpackTxCIDs,
	unpackStateRootCID,
	unpackResult,
	unpackFrames,
	unpackGas,
	unpackFailed,
}

func unpackTxCIDs(ma ipld.MapAssembler, txTrace TxTrace) error {
	if err := ma.AssembleKey().AssignString("TxCIDs"); err != nil {
		return err
	}
	la, err := ma.AssembleValue().BeginList(int64(len(txTrace.TxHashes)))
	if err != nil {
		return err
	}
	for _, txHash := range txTrace.TxHashes {
		txMh, err := multihash.Encode(txHash.Bytes(), tx.MultiHashType)
		if err != nil {
			return err
		}
		txCID := cid.NewCidV1(cid.EthTx, txMh)
		txLinkCID := cidlink.Link{Cid: txCID}
		if err := la.AssembleValue().AssignLink(txLinkCID); err != nil {
			return err
		}
	}
	return la.Finish()
}

func unpackStateRootCID(ma ipld.MapAssembler, txTrace TxTrace) error {
	srMh, err := multihash.Encode(txTrace.StateRoot.Bytes(), state_trie.MultiHashType)
	if err != nil {
		return err
	}
	srCID := cid.NewCidV1(cid.EthStateTrie, srMh)
	srLinkCID := cidlink.Link{Cid: srCID}
	if err := ma.AssembleKey().AssignString("StateRootCID"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignLink(srLinkCID)
}

func unpackResult(ma ipld.MapAssembler, txTrace TxTrace) error {
	if err := ma.AssembleKey().AssignString("Result"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(txTrace.Result)
}

func unpackFrames(ma ipld.MapAssembler, txTrace TxTrace) error {
	if err := ma.AssembleKey().AssignString("Frames"); err != nil {
		return err
	}
	framesLA, err := ma.AssembleValue().BeginList(int64(len(txTrace.Frames)))
	if err != nil {
		return err
	}
	for _, frame := range txTrace.Frames {
		frameMA, err := framesLA.AssembleValue().BeginMap(8)
		if err != nil {
			return err
		}
		if err := unpackFrame(frameMA, frame); err != nil {
			return err
		}
		if err := frameMA.Finish(); err != nil {
			return err
		}
	}
	return framesLA.Finish()
}

func unpackGas(ma ipld.MapAssembler, txTrace TxTrace) error {
	gasBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(gasBytes, txTrace.Gas)
	if err := ma.AssembleKey().AssignString("Gas"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(gasBytes)
}

func unpackFailed(ma ipld.MapAssembler, txTrace TxTrace) error {
	if err := ma.AssembleKey().AssignString("Failed"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBool(txTrace.Failed)
}

func unpackFrame(ma ipld.MapAssembler, frame Frame) error {
	for _, pFunc := range requiredFrameUnpackFuncs {
		if err := pFunc(ma, frame); err != nil {
			return err
		}
	}
	return nil
}

var requiredFrameUnpackFuncs = []func(ipld.MapAssembler, Frame) error{
	unpackOp,
	unpackFrom,
	unpackTo,
	unpackInput,
	unpackOutput,
	unpackFrameGas,
	unpackCost,
	unpackValue,
}

func unpackOp(ma ipld.MapAssembler, frame Frame) error {
	if err := ma.AssembleKey().AssignString("Op"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes([]byte{byte(frame.Op)})
}

func unpackFrom(ma ipld.MapAssembler, frame Frame) error {
	if err := ma.AssembleKey().AssignString("From"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(frame.From.Bytes())
}

func unpackTo(ma ipld.MapAssembler, frame Frame) error {
	if err := ma.AssembleKey().AssignString("To"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(frame.To.Bytes())
}

func unpackInput(ma ipld.MapAssembler, frame Frame) error {
	if err := ma.AssembleKey().AssignString("Input"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(frame.Input)
}

func unpackOutput(ma ipld.MapAssembler, frame Frame) error {
	if err := ma.AssembleKey().AssignString("Output"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(frame.Output)
}

func unpackFrameGas(ma ipld.MapAssembler, frame Frame) error {
	gasBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(gasBytes, frame.Gas)
	if err := ma.AssembleKey().AssignString("Gas"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(gasBytes)
}

func unpackCost(ma ipld.MapAssembler, frame Frame) error {
	costBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(costBytes, frame.Cost)
	if err := ma.AssembleKey().AssignString("Cost"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(costBytes)
}

func unpackValue(ma ipld.MapAssembler, frame Frame) error {
	if err := ma.AssembleKey().AssignString("Value"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(frame.Value.Bytes())
}
