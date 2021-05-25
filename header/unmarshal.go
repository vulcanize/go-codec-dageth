package dageth_header

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"
)

// Decode provides an IPLD codec decode interface for ETH header IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x90 when this package is invoked via init.
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
	var header types.Header
	if err := rlp.DecodeBytes(src, &header); err != nil {
		return err
	}
	return DecodeHeader(na, header)
}

// DecodeHeader unpacks a go-ethereum Header into a NodeAssembler
func DecodeHeader(na ipld.NodeAssembler, header types.Header) error {
	ma, err := na.BeginMap(15)
	if err != nil {
		return err
	}
	for _, upFunc := range requiredUnpackFuncs {
		if err := upFunc(ma, header); err != nil {
			return fmt.Errorf("invalid DAG-ETH Header binary (%v)", err)
		}
	}
	return ma.Finish()
}

var requiredUnpackFuncs = []func(ipld.MapAssembler, types.Header) error{
	unpackParentCID,
	unpackUnclesCID,
	unpackCoinbase,
	unpackStateRootCID,
	unpackTxRootCID,
	unpackRctRootCID,
	unpackBloom,
	unpackDifficulty,
	unpackNumber,
	unpackGasLimit,
	unpackGasUsed,
	unpackTime,
	unpackExtra,
	unpackMixDigest,
	unpackNonce,
}

func unpackNonce(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("Nonce"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.Nonce[:]); err != nil {
		return err
	}
	return nil
}

func unpackMixDigest(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("MixDigest"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.MixDigest.Bytes()); err != nil {
		return err
	}
	return nil
}

func unpackExtra(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("Extra"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(header.Extra)
}

func unpackTime(ma ipld.MapAssembler, header types.Header) error {
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, header.Time)
	if err := ma.AssembleKey().AssignString("Time"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(timeBytes)
}

func unpackGasUsed(ma ipld.MapAssembler, header types.Header) error {
	gasUsedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(gasUsedBytes, header.GasUsed)
	if err := ma.AssembleKey().AssignString("GasUsed"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(gasUsedBytes)
}

func unpackGasLimit(ma ipld.MapAssembler, header types.Header) error {
	gasLimitBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(gasLimitBytes, header.GasLimit)
	if err := ma.AssembleKey().AssignString("GasLimit"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(gasLimitBytes)
}

func unpackNumber(ma ipld.MapAssembler, header types.Header) error {
	if header.Number == nil {
		return fmt.Errorf("header cannot have `nil` Number")
	}
	if err := ma.AssembleKey().AssignString("Number"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(header.Number.Bytes())
}

func unpackDifficulty(ma ipld.MapAssembler, header types.Header) error {
	if header.Difficulty == nil {
		return fmt.Errorf("header cannot have `nil` Difficulty")
	}
	if err := ma.AssembleKey().AssignString("Difficulty"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(header.Difficulty.Bytes())
}

func unpackBloom(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("Bloom"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(header.Bloom.Bytes())
}

func unpackRctRootCID(ma ipld.MapAssembler, header types.Header) error {
	rctMh, err := multihash.Encode(header.ReceiptHash.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	rctCID := cid.NewCidV1(cid.EthTxReceipt, rctMh)
	rctLinkCID := cidlink.Link{Cid: rctCID}
	if err := ma.AssembleKey().AssignString("RctRootCID"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignLink(rctLinkCID)
}

func unpackTxRootCID(ma ipld.MapAssembler, header types.Header) error {
	txMh, err := multihash.Encode(header.TxHash.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	txCID := cid.NewCidV1(cid.EthTx, txMh)
	txLinkCID := cidlink.Link{Cid: txCID}
	if err := ma.AssembleKey().AssignString("TxRootCID"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignLink(txLinkCID)
}

func unpackStateRootCID(ma ipld.MapAssembler, header types.Header) error {
	srMh, err := multihash.Encode(header.Root.Bytes(), MultiHashType)
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

func unpackCoinbase(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("Coinbase"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(header.Coinbase.Bytes())
}

func unpackUnclesCID(ma ipld.MapAssembler, header types.Header) error {
	unclesMh, err := multihash.Encode(header.UncleHash.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	unclesCID := cid.NewCidV1(cid.EthBlockList, unclesMh)
	unclesLinkCID := cidlink.Link{Cid: unclesCID}
	if err := ma.AssembleKey().AssignString("UnclesCID"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignLink(unclesLinkCID)
}

func unpackParentCID(ma ipld.MapAssembler, header types.Header) error {
	parentMh, err := multihash.Encode(header.ParentHash.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	parentCID := cid.NewCidV1(cid.EthBlock, parentMh)
	parentLinkCID := cidlink.Link{Cid: parentCID}
	if err := ma.AssembleKey().AssignString("ParentCID"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignLink(parentLinkCID)
}
