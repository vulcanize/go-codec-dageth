package dageth_txlist

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
	dageth_tx "github.com/vulcanize/go-codec-dageth/tx"
)

// Encode provides an IPLD codec encode interface for eth transaction list IPLDs.
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
	txs := make([]*types.Transaction, 0, inNode.Length())
	if err := EncodeTxs(&txs, inNode); err != nil {
		return enc, err
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	if err := rlp.Encode(wbs, txs); err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Transactions form (unable to RLP encode transactions: %v)", err)
	}
	return enc, nil
}

// EncodeTxs packs the node into a go-ethereum Transactions
func EncodeTxs(txs *[]*types.Transaction, inNode ipld.Node) error {
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Transactions.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return err
	}
	node := builder.Build()
	txsIt := node.ListIterator()
	for !txsIt.Done() {
		_, txNode, err := txsIt.Next()
		if err != nil {
			return err
		}
		tx := new(types.Transaction)
		if err := dageth_tx.EncodeTx(tx, txNode); err != nil {
			return fmt.Errorf("invalid DAG-ETH Transactions form (%v)", err)
		}
		*txs = append(*txs, tx)
	}
	return nil
}
