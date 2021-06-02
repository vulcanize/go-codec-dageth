package dageth_txtrie

import (
	"io"

	"github.com/ipld/go-ipld-prime"

	dageth_trie "github.com/vulcanize/go-codec-dageth/trie"
)

// Encode provides an IPLD codec encode interface for eth tx trie node IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x92 when this package is invoked via init.
// This is a pure wrapping around dageth_trie.Encode to expose it from this package
func Encode(node ipld.Node, w io.Writer) error {
	return dageth_trie.Encode(node, w)
}

// AppendEncode is like Encode, but it uses a destination buffer directly.
// This means less copying of bytes, and if the destination has enough capacity,
// fewer allocations.
// This is a pure wrapping around dageth_trie.AppendEncode to expose it from this package
func AppendEncode(enc []byte, inNode ipld.Node) ([]byte, error) {
	return dageth_trie.AppendEncode(enc, inNode)
}
