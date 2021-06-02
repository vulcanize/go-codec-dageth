package dageth_trie

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	dageth_rct "github.com/vulcanize/go-codec-dageth/rct"
	"github.com/vulcanize/go-codec-dageth/shared"
	dageth_account "github.com/vulcanize/go-codec-dageth/state_account"
	dageth_tx "github.com/vulcanize/go-codec-dageth/tx"
)

type NodeKind string
type ValueKind string

const (
	UNKNOWN_NODE   NodeKind = "unknown"
	BRANCH_NODE    NodeKind = "branch"
	EXTENSION_NODE NodeKind = "extension"
	LEAF_NODE      NodeKind = "leaf"

	UNKNOWN_VALUE ValueKind = "unknown"
	TX_VALUE      ValueKind = "tx"
	RCT_VALUE     ValueKind = "rct"
	STATE_VALUE   ValueKind = "state"
	STORAGE_VALUE ValueKind = "storage"
)

func (n NodeKind) String() string {
	return string(n)
}

func (v ValueKind) String() string {
	return string(v)
}

// Encode provides an IPLD codec encode interface for eth merkle patricia trie node IPLDs.
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
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.TrieNode.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return nil, err
	}
	n := builder.Build()
	node, kind, err := NodeAndKind(n)
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
		case ipld.Kind_Map:
			// it must be a leaf node as only RLP encodings of storage leaf nodes can be less than 32 bytes in length and stored direclty in a parent node
			childLeafNode, err := childNode.LookupByString(LEAF_NODE.String())
			if err != nil {
				return nil, fmt.Errorf("only leaf nodes can be less than 32 bytes and stored direclty in a parent node")
			}
			childLeafNodeFields, err := packLeafNode(childLeafNode)
			if err != nil {
				return nil, err
			}
			childLeafNodeRLP, err := rlp.EncodeToBytes(childLeafNodeFields)
			if err != nil {
				return nil, err
			}
			nodeFields[i] = childLeafNodeRLP
		default:
			return nil, fmt.Errorf("branch node child needs to be of kind bytes, link, or null")
		}
	}
	valueBytes, err := packValue(node)
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
	nodeFields[0] = pp // TODO: need to compact the key
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
	case ipld.Kind_Map: // is this possible? Will an extension node ever link to a leaf? In that case it could just be a leaf itself...?
		// it must be a leaf node as only RLP encodings of storage leaf nodes can be less than 32 bytes in length and stored direclty in a parent node
		childLeafNode, err := childNode.LookupByString(LEAF_NODE.String())
		if err != nil {
			return nil, fmt.Errorf("only leaf nodes can be less than 32 bytes and stored direclty in a parent node")
		}
		childLeafNodeFields, err := packLeafNode(childLeafNode)
		if err != nil {
			return nil, err
		}
		childLeafNodeRLP, err := rlp.EncodeToBytes(childLeafNodeFields)
		if err != nil {
			return nil, err
		}
		nodeFields[1] = childLeafNodeRLP
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
	nodeFields[0] = pp // TODO: need to compact the key
	valueBytes, err := packValue(node)
	if err != nil {
		return nil, err
	}
	nodeFields[1] = valueBytes
	return nodeFields, nil
}

func packValue(node ipld.Node) ([]byte, error) {
	valUnionNode, err := node.LookupByString("Value")
	if err != nil {
		return nil, err
	}
	if valUnionNode.IsNull() {
		return []byte{}, nil
	}
	valNode, valKind, err := ValueAndKind(valUnionNode)
	if err != nil {
		return nil, err
	}
	switch valKind {
	case TX_VALUE:
		buf := new(bytes.Buffer)
		if err := dageth_tx.Encode(valNode, buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case RCT_VALUE:
		buf := new(bytes.Buffer)
		if err := dageth_rct.Encode(valNode, buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case STATE_VALUE:
		buf := new(bytes.Buffer)
		if err := dageth_account.Encode(valNode, buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case STORAGE_VALUE:
		return valNode.AsBytes()
	default:
		return nil, fmt.Errorf("eth trie value of unexpected kind %s", valKind.String())
	}
}

func ValueAndKind(node ipld.Node) (ipld.Node, ValueKind, error) {
	n, err := node.LookupByString(TX_VALUE.String())
	if err == nil {
		return n, TX_VALUE, nil
	}
	n, err = node.LookupByString(RCT_VALUE.String())
	if err == nil {
		return n, RCT_VALUE, nil
	}
	n, err = node.LookupByString(STATE_VALUE.String())
	if err == nil {
		return n, STATE_VALUE, nil
	}
	n, err = node.LookupByString(STORAGE_VALUE.String())
	if err == nil {
		return n, STORAGE_VALUE, nil
	}
	return nil, "", fmt.Errorf("eth trie value IPLD node is missing the expected keyed Union keys")
}

func NodeAndKind(node ipld.Node) (ipld.Node, NodeKind, error) {
	n, err := node.LookupByString(LEAF_NODE.String())
	if err == nil {
		return n, LEAF_NODE, nil
	}
	n, err = node.LookupByString(BRANCH_NODE.String())
	if err == nil {
		return n, BRANCH_NODE, nil
	}
	n, err = node.LookupByString(EXTENSION_NODE.String())
	if err == nil {
		return n, EXTENSION_NODE, nil
	}
	return nil, "", fmt.Errorf("eth trie node IPLD node is missing the expected keyed Union keys")
}
