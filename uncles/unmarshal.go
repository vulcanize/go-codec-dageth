package uncles

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth_header "github.com/vulcanize/go-codec-dageth/header"
)

// Decode provides an IPLD codec decode interface for eth uncles IPLDs (header list).
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x91 when this package is invoked via init.
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
	var uncles []*types.Header
	if err := rlp.DecodeBytes(src, &uncles); err != nil {
		return err
	}

	return DecodeUncles(na, uncles)
}

// DecodeUncles unpacks a list of go-ethereum headers into the NodeAssembler
func DecodeUncles(na ipld.NodeAssembler, uncles []*types.Header) error {
	la, err := na.BeginList(int64(len(uncles)))
	if err != nil {
		return err
	}
	for i, uncle := range uncles {
		// node := dageth.Type.Header.NewBuilder()
		node := la.ValuePrototype(int64(i)).NewBuilder()
		if err := dageth_header.DecodeHeader(node, *uncle); err != nil {
			return fmt.Errorf("invalid DAG-ETH Uncles binary (%v)", err)
		}
		if err := la.AssembleValue().AssignNode(node.Build()); err != nil {
			return err
		}
	}
	return la.Finish()
}
