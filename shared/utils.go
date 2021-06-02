package shared

import (
	"fmt"

	"github.com/ipld/go-ipld-prime"
)

func GetTxType(node ipld.Node) (uint8, error) {
	tyNode, err := node.LookupByString("Type")
	if err != nil {
		return 0, err
	}
	tyBytes, err := tyNode.AsBytes()
	if err != nil {
		return 0, err
	}
	if len(tyBytes) != 1 {
		return 0, fmt.Errorf("tx type should be a single byte")
	}
	return tyBytes[0], nil
}

type WriteableByteSlice []byte

func (w *WriteableByteSlice) Write(b []byte) (int, error) {
	*w = append(*w, b...)
	return len(b), nil
}
