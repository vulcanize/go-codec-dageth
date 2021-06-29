package header

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
)

// Encode provides an IPLD codec encode interface for eth header IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x90 when this package is invoked via init.
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
	header := new(types.Header)
	if err := EncodeHeader(header, inNode); err != nil {
		return enc, err
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	if err := rlp.Encode(wbs, header); err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Header form (unable to RLP encode header: %v)", err)
	}
	return enc, nil
}

// EncodeHeader packs the node into the provided go-ethereum Header
func EncodeHeader(header *types.Header, inNode ipld.Node) error {
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Header.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return err
	}
	node := builder.Build()
	for _, pFunc := range requiredPackFuncs {
		if err := pFunc(header, node); err != nil {
			return fmt.Errorf("invalid DAG-ETH Header form (%v)", err)
		}
	}
	return nil
}

var requiredPackFuncs = []func(*types.Header, ipld.Node) error{
	packParentCID,
	packUnclesCID,
	packCoinbase,
	packStateRootCID,
	packTxRootCID,
	packRctRootCID,
	packBloom,
	packDifficulty,
	packNumber,
	packGasLimit,
	packGasUsed,
	packTime,
	packExtra,
	packMixDigest,
	packNonce,
	packBaseFee,
}

func packNonce(header *types.Header, node ipld.Node) error {
	n, err := node.LookupByString("Nonce")
	if err != nil {
		return err
	}
	nBytes, err := n.AsBytes()
	if err != nil {
		return err
	}
	if len(nBytes) != len(types.BlockNonce{}) {
		return fmt.Errorf("header must have a %d byte Nonce", len(types.BlockNonce{}))
	}
	copy(header.Nonce[:], nBytes)
	return nil
}

func packMixDigest(header *types.Header, node ipld.Node) error {
	md, err := node.LookupByString("MixDigest")
	if err != nil {
		return err
	}
	mdBytes, err := md.AsBytes()
	if err != nil {
		return err
	}
	header.MixDigest = common.BytesToHash(mdBytes)
	return nil
}

func packExtra(header *types.Header, node ipld.Node) error {
	e, err := node.LookupByString("Extra")
	if err != nil {
		return err
	}
	extra, err := e.AsBytes()
	if err != nil {
		return err
	}
	header.Extra = extra
	return nil
}

func packTime(header *types.Header, node ipld.Node) error {
	t, err := node.LookupByString("Time")
	if err != nil {
		return err
	}
	tBytes, err := t.AsBytes()
	if err != nil {
		return err
	}
	header.Time = binary.BigEndian.Uint64(tBytes)
	return nil
}

func packGasUsed(header *types.Header, node ipld.Node) error {
	gu, err := node.LookupByString("GasUsed")
	if err != nil {
		return err
	}
	guBytes, err := gu.AsBytes()
	if err != nil {
		return err
	}
	header.GasUsed = binary.BigEndian.Uint64(guBytes)
	return nil
}

func packGasLimit(header *types.Header, node ipld.Node) error {
	gl, err := node.LookupByString("GasLimit")
	if err != nil {
		return err
	}
	glBytes, err := gl.AsBytes()
	if err != nil {
		return err
	}
	header.GasLimit = binary.BigEndian.Uint64(glBytes)
	return nil
}

func packNumber(header *types.Header, node ipld.Node) error {
	num, err := node.LookupByString("Number")
	if err != nil {
		return err
	}
	numBytes, err := num.AsBytes()
	if err != nil {
		return err
	}
	header.Number = new(big.Int).SetBytes(numBytes)
	return nil
}

func packDifficulty(header *types.Header, node ipld.Node) error {
	diff, err := node.LookupByString("Difficulty")
	if err != nil {
		return err
	}
	diffBytes, err := diff.AsBytes()
	if err != nil {
		return err
	}
	header.Difficulty = new(big.Int).SetBytes(diffBytes)
	return nil
}

func packBloom(header *types.Header, node ipld.Node) error {
	blm, err := node.LookupByString("Bloom")
	if err != nil {
		return err
	}
	blmBytes, err := blm.AsBytes()
	if err != nil {
		return err
	}
	// prevent any chance of BytesToBloom panicing on wrong bytes length
	if len(blmBytes) != types.BloomByteLength {
		return fmt.Errorf("header must have a 256 byte Bloom")
	}
	header.Bloom = types.BytesToBloom(blmBytes)
	return nil
}

func packRctRootCID(header *types.Header, node ipld.Node) error {
	rctCID, err := node.LookupByString("RctRootCID")
	if err != nil {
		return err
	}
	rctLink, err := rctCID.AsLink()
	if err != nil {
		return err
	}
	rctCIDLink, ok := rctLink.(cidlink.Link)
	if !ok {
		return fmt.Errorf("header must have a RctRootCID")
	}
	rctMh := rctCIDLink.Hash()
	decodedRctMh, err := multihash.Decode(rctMh)
	if err != nil {
		return fmt.Errorf("unable to decode RctRootCID multihash: %v", err)
	}
	header.ReceiptHash = common.BytesToHash(decodedRctMh.Digest)
	return nil
}

func packTxRootCID(header *types.Header, node ipld.Node) error {
	txCID, err := node.LookupByString("TxRootCID")
	if err != nil {
		return err
	}
	txLink, err := txCID.AsLink()
	if err != nil {
		return err
	}
	txCIDLink, ok := txLink.(cidlink.Link)
	if !ok {
		return fmt.Errorf("header must have a TxRootCID")
	}
	txMh := txCIDLink.Hash()
	decodedTxMh, err := multihash.Decode(txMh)
	if err != nil {
		return fmt.Errorf("unable to decode TxRootCID multihash: %v", err)
	}
	header.TxHash = common.BytesToHash(decodedTxMh.Digest)
	return nil
}

func packStateRootCID(header *types.Header, node ipld.Node) error {
	srCID, err := node.LookupByString("StateRootCID")
	if err != nil {
		return err
	}
	srLink, err := srCID.AsLink()
	if err != nil {
		return err
	}
	srCIDLink, ok := srLink.(cidlink.Link)
	if !ok {
		return fmt.Errorf("header must have a StateRootCID")
	}
	srMh := srCIDLink.Hash()
	decodedSrMh, err := multihash.Decode(srMh)
	if err != nil {
		return fmt.Errorf("unable to decode StateRootCID multihash: %v", err)
	}
	header.Root = common.BytesToHash(decodedSrMh.Digest)
	return nil
}

func packCoinbase(header *types.Header, node ipld.Node) error {
	coinbase, err := node.LookupByString("Coinbase")
	if err != nil {
		return err
	}
	coinbaseBytes, err := coinbase.AsBytes()
	if err != nil {
		return err
	}
	header.Coinbase = common.BytesToAddress(coinbaseBytes)
	return nil
}

func packUnclesCID(header *types.Header, node ipld.Node) error {
	uncleCID, err := node.LookupByString("UnclesCID")
	if err != nil {
		return err
	}
	unclesLink, err := uncleCID.AsLink()
	if err != nil {
		return err
	}
	unclesCIDLink, ok := unclesLink.(cidlink.Link)
	if !ok {
		return fmt.Errorf("header must have an UnclesCID")
	}
	unclesMh := unclesCIDLink.Hash()
	decodedUnclesMh, err := multihash.Decode(unclesMh)
	if err != nil {
		return fmt.Errorf("unable to decode UnclesCID multihash: %v", err)
	}
	header.UncleHash = common.BytesToHash(decodedUnclesMh.Digest)
	return nil
}

func packParentCID(header *types.Header, node ipld.Node) error {
	parentCID, err := node.LookupByString("ParentCID")
	if err != nil {
		return err
	}
	parentLink, err := parentCID.AsLink()
	if err != nil {
		return err
	}
	parentCIDLink, ok := parentLink.(cidlink.Link)
	if !ok {
		return fmt.Errorf("header must have a ParentCID")
	}
	parentMh := parentCIDLink.Hash()
	decodedParentMh, err := multihash.Decode(parentMh)
	if err != nil {
		return fmt.Errorf("unable to decode ParentCID multihash: %v", err)
	}
	header.ParentHash = common.BytesToHash(decodedParentMh.Digest)
	return nil
}

func packBaseFee(header *types.Header, node ipld.Node) error {
	baseFeeNode, err := node.LookupByString("BaseFee")
	if err != nil {
		return err
	}
	if baseFeeNode.IsNull() {
		return nil
	}
	baseFeeBytes, err := baseFeeNode.AsBytes()
	if err != nil {
		return err
	}
	header.BaseFee = new(big.Int).SetBytes(baseFeeBytes)
	return nil
}
