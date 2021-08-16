package tx_list

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth_tx "github.com/vulcanize/go-codec-dageth/tx"
)

// Decode provides an IPLD codec decode interface for eth transaction list IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code tbd when this package is invoked via init.
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
	var txs []*types.Transaction
	if err := rlp.DecodeBytes(src, &txs); err != nil {
		return err
	}

	return DecodeTxs(na, txs)
}

// DecodeTxs unpacks a list of go-ethereum Transactions into the NodeAssembler
func DecodeTxs(na ipld.NodeAssembler, txs []*types.Transaction) error {
	la, err := na.BeginList(int64(len(txs)))
	if err != nil {
		return err
	}
	for i, tx := range txs {
		// node := dageth.Type.Transaction.NewBuilder()
		node := la.ValuePrototype(int64(i)).NewBuilder()
		if err := dageth_tx.DecodeTx(node, *tx); err != nil {
			return fmt.Errorf("invalid DAG-ETH Transactions binary (%v)", err)
		}
		if err := la.AssembleValue().AssignNode(node.Build()); err != nil {
			return err
		}
	}
	return la.Finish()
}
