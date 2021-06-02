package dageth_trie

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	"github.com/vulcanize/go-codec-dageth/shared"
)

type NodeKind string

const (
	UNKNOWN_NODE   NodeKind = "Unknown"
	BRANCH_NODE    NodeKind = "TrieBranchNode"
	EXTENSION_NODE NodeKind = "TrieExtensionNode"
	LEAF_NODE      NodeKind = "TrieLeafNode"
)

func (n NodeKind) String() string {
	return string(n)
}

// Encode provides an IPLD codec encode interface for eth trie IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code XXXX when this package is invoked via init.
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
	node, kind, err := NodeAndKind(inNode)
	if err != nil {
		return nil, err
	}
	var nodeFields []interface{}
	switch kind {
	case BRANCH_NODE:
		nodeFields, err = packBranchNode(node)
		if err != nil {
			return nil, err
		}
	case EXTENSION_NODE:
		nodeFields, err = packExtensionNode(node)
		if err != nil {
			return nil, err
		}
	case LEAF_NODE:
		nodeFields, err = packLeafNode(node)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("IPLD node is missing the expected Union keys")
	}
	wbs := shared.WriteableByteSlice(enc)
	if err := rlp.Encode(&wbs, nodeFields); err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH TrieNode form (%v)", err)
	}
	return wbs, nil
}

func packBranchNode(node ipld.Node) ([]interface{}, error) {
	nodeFields := make([]interface{}, 17)
	for i := 0; i < 16; i++ {
		key := fmt.Sprintf("Child%d", i)
		childNode, err := node.LookupByString(key)
		if err != nil {
			return nil, err
		}
		switch childNode.Kind() {
		case ipld.Kind_Null:
			nodeFields[i] = []byte{}
		case ipld.Kind_Link:
			childLink, err := childNode.AsLink()
			if err != nil {
				return nil, err
			}
			childCIDLink, ok := childLink.(cidlink.Link)
			if !ok {
				return nil, fmt.Errorf("branch node child link needs to be a CID")
			}
			childMh := childCIDLink.Hash()
			decodedChildMh, err := multihash.Decode(childMh)
			if err != nil {
				return nil, fmt.Errorf("unable to decode Child multihash: %v", err)
			}
			nodeFields[i] = decodedChildMh.Digest
		case ipld.Kind_Map: // TODO:
			childBytes, err := childNode.AsBytes()
			if err != nil {
				return nil, err
			}
			nodeFields[i] = childBytes
		default:
			return nil, fmt.Errorf("branch node child needs to be of kind bytes, link, or null")
		}
	}
	valueNode, err := node.LookupByString("Value")
	if err != nil {
		return nil, err
	}
	if valueNode.IsNull() {
		nodeFields[16] = []byte{}
		return nil, err
	}
	valueBytes, err := valueNode.AsBytes()
	if err != nil {
		return nil, err
	}
	nodeFields[16] = valueBytes
	return nodeFields, nil
}

func packExtensionNode(node ipld.Node) ([]interface{}, error) {
	nodeFields := make([]interface{}, 2)
	ppNode, err := node.LookupByString("PartialPath")
	if err != nil {
		return nil, err
	}
	pp, err := ppNode.AsBytes()
	if err != nil {
		return nil, err
	}
	nodeFields[0] = pp
	childNode, err := node.LookupByString("Child")
	if err != nil {
		return nil, err
	}
	switch childNode.Kind() {
	case ipld.Kind_Link:
		childLink, err := childNode.AsLink()
		if err != nil {
			return nil, err
		}
		childCIDLink, ok := childLink.(cidlink.Link)
		if !ok {
			return nil, fmt.Errorf("extension node child link needs to be a CID")
		}
		childMh := childCIDLink.Hash()
		decodedChildMh, err := multihash.Decode(childMh)
		if err != nil {
			return nil, fmt.Errorf("unable to decode Child multihash: %v", err)
		}
		nodeFields[1] = decodedChildMh.Digest
	case ipld.Kind_Map: // TODO:
		childBytes, err := childNode.AsBytes()
		if err != nil {
			return nil, err
		}
		nodeFields[1] = childBytes
	default:
		return nil, fmt.Errorf("extension node child needs to be of kind bytes or link")
	}
	return nodeFields, nil
}

func packLeafNode(node ipld.Node) ([]interface{}, error) {
	nodeFields := make([]interface{}, 2)
	ppNode, err := node.LookupByString("PartialPath")
	if err != nil {
		return nil, err
	}
	pp, err := ppNode.AsBytes()
	if err != nil {
		return nil, err
	}
	nodeFields[0] = pp
	valNode, err := node.LookupByString("Value")
	if err != nil {
		return nil, err
	}
	val, err := valNode.AsBytes()
	if err != nil {
		return nil, err
	}
	nodeFields[1] = val
	return nodeFields, nil
}

func NodeAndKind(node ipld.Node) (ipld.Node, NodeKind, error) {
	n, err := node.LookupByString(BRANCH_NODE.String())
	if err == nil {
		return n, BRANCH_NODE, nil
	}
	n, err = node.LookupByString(EXTENSION_NODE.String())
	if err == nil {
		return n, EXTENSION_NODE, nil
	}
	n, err = node.LookupByString(LEAF_NODE.String())
	if err == nil {
		return n, LEAF_NODE, nil
	}
	return nil, "", fmt.Errorf("IPLD node is missing the expected keyed Union keys")
}
