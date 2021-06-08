package state_trie

import (
	"io"

	"github.com/ipld/go-ipld-prime"

	dageth_trie "github.com/vulcanize/go-codec-dageth/trie"
)

// Decode provides an IPLD codec decode interface for eth state trie node IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x96 when this package is invoked via init.
// This simply wraps dageth_trie.DecodeTrieNode with the proper multicodec type
func Decode(na ipld.NodeAssembler, in io.Reader) error {
	return dageth_trie.DecodeTrieNode(na, in, MultiCodecType)
}

// DecodeBytes is like Decode, but it uses an input buffer directly.
// Decode will grab or read all the bytes from an io.Reader anyway, so this can
// save having to copy the bytes or create a bytes.Buffer.
// This simply wraps dageth_trie.DecodeTrieNodeBytes with the proper multicodec type
func DecodeBytes(na ipld.NodeAssembler, src []byte) error {
	return dageth_trie.DecodeTrieNodeBytes(na, src, MultiCodecType)
}
