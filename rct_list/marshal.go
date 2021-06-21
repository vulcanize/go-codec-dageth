package rct_list

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	dageth_rct "github.com/vulcanize/go-codec-dageth/rct"
	"github.com/vulcanize/go-codec-dageth/shared"
)

// Encode provides an IPLD codec encode interface for eth receipt list IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code (tbd) when this package is invoked via init.
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
	rcts := make([]*types.Receipt, 0, inNode.Length())
	if err := EncodeRcts(&rcts, inNode); err != nil {
		return enc, err
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	if err := rlp.Encode(wbs, rcts); err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Receipts form (unable to RLP encode receipts: %v)", err)
	}
	return enc, nil
}

// EncodeRcts packs the node into a go-ethereum Receipts
func EncodeRcts(rcts *[]*types.Receipt, inNode ipld.Node) error {
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Receipts.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return err
	}
	node := builder.Build()
	rctsIt := node.ListIterator()
	for !rctsIt.Done() {
		_, rctNode, err := rctsIt.Next()
		if err != nil {
			return err
		}
		rct := new(types.Receipt)
		if err := dageth_rct.EncodeReceipt(rct, rctNode); err != nil {
			return fmt.Errorf("invalid DAG-ETH Receipts form (%v)", err)
		}
		*rcts = append(*rcts, rct)
	}
	return nil
}
