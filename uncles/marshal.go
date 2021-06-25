package uncles

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	dageth_header "github.com/vulcanize/go-codec-dageth/header"
	"github.com/vulcanize/go-codec-dageth/shared"
)

// Encode provides an IPLD codec encode interface for eth uncles IPLDs (header list).
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x91 when this package is invoked via init.
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
	uncles := make([]*types.Header, 0, inNode.Length())
	if err := EncodeUncles(&uncles, inNode); err != nil {
		return enc, err
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	if err := rlp.Encode(wbs, uncles); err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Uncles form (unable to RLP encode uncles: %v)", err)
	}
	return enc, nil
}

// EncodeUncles packs the node into a list of go-ethereum headers
func EncodeUncles(uncles *[]*types.Header, inNode ipld.Node) error {
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Uncles.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return err
	}
	node := builder.Build()
	unclesIt := node.ListIterator()
	for !unclesIt.Done() {
		_, uncleNode, err := unclesIt.Next()
		if err != nil {
			return err
		}
		uncle := new(types.Header)
		if err := dageth_header.EncodeHeader(uncle, uncleNode); err != nil {
			return fmt.Errorf("invalid DAG-ETH Uncles form (%v)", err)
		}
		*uncles = append(*uncles, uncle)
	}
	return nil
}
