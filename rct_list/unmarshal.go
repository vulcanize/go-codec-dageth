package rct_list

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth_rct "github.com/vulcanize/go-codec-dageth/rct"
)

// Decode provides an IPLD codec decode interface for eth receipt list IPLDs.
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
	var rcts []*types.Receipt
	if err := rlp.DecodeBytes(src, &rcts); err != nil {
		return err
	}

	return DecodeRcts(na, rcts)
}

// DecodeRcts unpacks a list of go-ethereum Receipts into the NodeAssembler
func DecodeRcts(na ipld.NodeAssembler, rcts []*types.Receipt) error {
	la, err := na.BeginList(int64(len(rcts)))
	if err != nil {
		return err
	}
	for i, rct := range rcts {
		// node := dageth.Type.Receipt.NewBuilder()
		node := la.ValuePrototype(int64(i)).NewBuilder()
		if err := dageth_rct.DecodeReceipt(node, *rct); err != nil {
			return fmt.Errorf("invalid DAG-ETH Receipts binary (%v)", err)
		}
		la.AssembleValue().AssignNode(node.Build())
	}
	return la.Finish()
}
