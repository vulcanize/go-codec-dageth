package dageth_uncles

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
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Uncles.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return enc, err
	}
	node := builder.Build()
	uncles := make([]*types.Header, node.Length())
	unclesIt := node.ListIterator()
	for !unclesIt.Done() {
		index, uncleNode, err := unclesIt.Next()
		if err != nil {
			return enc, err
		}
		uncle := new(types.Header)
		for _, pFunc := range dageth_header.RequiredPackFuncs {
			if err := pFunc(uncle, uncleNode); err != nil {
				return enc, fmt.Errorf("invalid DAG-ETH Uncles form (%v)", err)
			}
		}
		uncles[index] = uncle
	}
	wbs := shared.WriteableByteSlice(enc)
	if err := rlp.Encode(&wbs, uncles); err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Uncles form (unable to RLP encode uncles: %v)", err)
	}
	return enc, nil
}
