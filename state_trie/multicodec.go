package state_trie

import (
	"io"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/multicodec"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
)

var (
	_ ipld.Decoder = Decode
	_ ipld.Encoder = Encode

	MultiCodecType = uint64(cid.EthStateTrie) // 0x96
	MultiHashType  = uint64(multihash.KECCAK_256)
)

func init() {
	multicodec.RegisterDecoder(MultiCodecType, Decode)
	multicodec.RegisterEncoder(MultiCodecType, Encode)
}

// AddSupportToChooser takes an existing node prototype chooser and subs in
// TrieNode for the eth state trie multicodec code.
func AddSupportToChooser(existing traversal.LinkTargetNodePrototypeChooser) traversal.LinkTargetNodePrototypeChooser {
	return func(lnk ipld.Link, lnkCtx ipld.LinkContext) (ipld.NodePrototype, error) {
		if lnk, ok := lnk.(cidlink.Link); ok && lnk.Cid.Prefix().Codec == MultiCodecType {
			return dageth.Type.TrieNode, nil
		}
		return existing(lnk, lnkCtx)
	}
}

// We switched to simpler API names after v1.0.0, so keep the old names around
// as deprecated forwarding funcs until a future v2+.
// TODO: consider deprecating Marshal/Unmarshal too, since it's a bit
// unnecessary to have two supported names for each API.

// Deprecated: use Decode instead.
func Decoder(na ipld.NodeAssembler, r io.Reader) error { return Decode(na, r) }

// Deprecated: use Decode instead.
func Unmarshal(na ipld.NodeAssembler, r io.Reader) error { return Decode(na, r) }

// Deprecated: use Encode instead.
func Encoder(inNode ipld.Node, w io.Writer) error { return Encode(inNode, w) }

// Deprecated: use Encode instead.
func Marshal(inNode ipld.Node, w io.Writer) error { return Encode(inNode, w) }
