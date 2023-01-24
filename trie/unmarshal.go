package trie

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/vulcanize/go-codec-dageth/log"
	"github.com/vulcanize/go-codec-dageth/rct"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
	account "github.com/vulcanize/go-codec-dageth/state_account"
	"github.com/vulcanize/go-codec-dageth/tx"
)

const logTrieMulticodec = uint64(0x99) // Proposed

// DecodeTrieNode provides an IPLD codec decode interface for eth merkle patricia trie nodes
// It's not possible to meet the Decode(na ipld.NodeAssembler, in io.Reader) interface
// for a function that supports all trie types (multicodec types), unlike with encoding.
// this is used by Decode functions for each trie type, which are the ones registered to their
// corresponding multicodec
func DecodeTrieNode(na ipld.NodeAssembler, in io.Reader, codec uint64) error {
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
	return DecodeTrieNodeBytes(na, src, codec)
}

// DecodeTrieNodeBytes is like DecodeTrieNode, but it uses an input buffer directly.
func DecodeTrieNodeBytes(na ipld.NodeAssembler, src []byte, codec uint64) error {
	var nodeFields []interface{}
	if err := rlp.DecodeBytes(src, &nodeFields); err != nil {
		return err
	}
	ma, err := na.BeginMap(1)
	if err != nil {
		return err
	}
	switch len(nodeFields) {
	case 2:
		nodeKind, decoded, err := decodeTwoMemberNode(nodeFields)
		if err != nil {
			return err
		}
		switch nodeKind {
		case EXTENSION_NODE:
			if err := ma.AssembleKey().AssignString(EXTENSION_NODE.String()); err != nil {
				return err
			}
			extNodeMA, err := ma.AssembleValue().BeginMap(2)
			if err != nil {
				return err
			}
			if err := unpackExtensionNode(extNodeMA, decoded, codec); err != nil {
				return err
			}
			if err := extNodeMA.Finish(); err != nil {
				return err
			}
		case LEAF_NODE:
			if err := ma.AssembleKey().AssignString(LEAF_NODE.String()); err != nil {
				return err
			}
			leafNodeMA, err := ma.AssembleValue().BeginMap(2)
			if err != nil {
				return err
			}
			if err := unpackLeafNode(leafNodeMA, decoded, codec); err != nil {
				return err
			}
			if err := leafNodeMA.Finish(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unrecognized trie node type %s", nodeKind.String())
		}
	case 17:
		if err := ma.AssembleKey().AssignString(BRANCH_NODE.String()); err != nil {
			return err
		}
		branchNodeMA, err := ma.AssembleValue().BeginMap(17)
		if err != nil {
			return err
		}
		if err := unpackBranchNode(branchNodeMA, nodeFields, codec); err != nil {
			return err
		}
		if err := branchNodeMA.Finish(); err != nil {
			return err
		}
	}
	return ma.Finish()
}

func unpackExtensionNode(ma ipld.MapAssembler, nodeFields []interface{}, codec uint64) error {
	partialPath, ok := nodeFields[0].([]byte)
	if !ok {
		return fmt.Errorf("extension node requires partial path byte slice")
	}
	if err := ma.AssembleKey().AssignString("PartialPath"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(partialPath); err != nil { // de-compact partial path first?
		return err
	}
	if err := ma.AssembleKey().AssignString("Child"); err != nil {
		return err
	}
	childLink, ok := nodeFields[1].([]byte)
	if !ok {
		return fmt.Errorf("unable to assert second member of extension node to type `[]byte`")
	}
	childCID := shared.Keccak256ToCid(codec, childLink)
	childCIDLink := cidlink.Link{Cid: childCID}
	return ma.AssembleValue().AssignLink(childCIDLink)
}

func unpackBranchNode(ma ipld.MapAssembler, nodeFields []interface{}, codec uint64) error {
	for i := 0; i < 16; i++ {
		key := fmt.Sprintf("Child%s", strings.ToUpper(strconv.FormatInt(int64(i), 16)))
		if err := ma.AssembleKey().AssignString(key); err != nil {
			return err
		}
		childNodeBuilder := dageth.Type.Child.NewBuilder()
		childNodeMA, err := childNodeBuilder.BeginMap(1)
		if err != nil {
			return err
		}
		childLink, ok := nodeFields[i].([]byte)
		if ok {
			switch len(childLink) {
			case 0:
				if err := ma.AssembleValue().AssignNull(); err != nil {
					return err
				}
			case 32:
				// it's a hash referencing the child node
				// make CID link from the bytes
				// assign the link value to the MA
				childCID := shared.Keccak256ToCid(codec, childLink)
				childCIDLink := cidlink.Link{Cid: childCID}
				if err := childNodeMA.AssembleKey().AssignString("Link"); err != nil {
					return err
				}
				if err := childNodeMA.AssembleValue().AssignLink(childCIDLink); err != nil {
					return err
				}
				if err := childNodeMA.Finish(); err != nil {
					return err
				}
				if err := ma.AssembleValue().AssignNode(childNodeBuilder.Build()); err != nil {
					return err
				}
			default:
				return fmt.Errorf("branch node child (%d) of unexpected length %d", i, len(childLink))
			}
			continue
		}
		// the child node is included directly
		// it must be a leaf node, branch and extension will never be less than 32 bytes
		childLeaf, ok := nodeFields[i].([]interface{})
		if !ok {
			return fmt.Errorf("unable to decode branch node entry into []byte or []interface{}")
		}
		if len(childLeaf) != 2 {
			return fmt.Errorf("unexpected number of entries for leaf node; got %d want 2", len(childLeaf))
		}
		nodeKind, decodedChildLeaf, err := decodeTwoMemberNode(childLeaf)
		if err != nil {
			return err
		}
		if nodeKind != LEAF_NODE {
			return fmt.Errorf("child node included directly in branch must be a leaf; got %s", nodeKind.String())
		}
		if err := childNodeMA.AssembleKey().AssignString("TrieNode"); err != nil {
			return err
		}
		childTrieNodeMA, err := childNodeMA.AssembleValue().BeginMap(1)
		if err != nil {
			return err
		}
		if err := childTrieNodeMA.AssembleKey().AssignString(LEAF_NODE.String()); err != nil {
			return err
		}
		leafNodeMA, err := childTrieNodeMA.AssembleValue().BeginMap(2)
		if err != nil {
			return err
		}
		if err := unpackLeafNode(leafNodeMA, decodedChildLeaf, codec); err != nil {
			return err
		}
		if err := leafNodeMA.Finish(); err != nil {
			return err
		}
		if err := childTrieNodeMA.Finish(); err != nil {
			return err
		}
		if err := childNodeMA.Finish(); err != nil {
			return err
		}
		if err := ma.AssembleValue().AssignNode(childNodeBuilder.Build()); err != nil {
			return err
		}
	}
	if err := ma.AssembleKey().AssignString("Value"); err != nil {
		return err
	}
	valBytes, ok := nodeFields[16].([]byte)
	if !ok {
		return fmt.Errorf("branch node 17th member should be a byte array (val)")
	}
	if len(valBytes) == 0 {
		return ma.AssembleValue().AssignNull()
	}
	valUnionNodeMA, err := ma.AssembleValue().BeginMap(1)
	if err != nil {
		return err
	}
	if err := unpackValue(valUnionNodeMA, valBytes, codec); err != nil {
		return err
	}
	return valUnionNodeMA.Finish()
}

func unpackLeafNode(ma ipld.MapAssembler, nodeFields []interface{}, codec uint64) error {
	partialPath, ok := nodeFields[0].([]byte)
	if !ok {
		return fmt.Errorf("leaf node requires partial path byte slice")
	}
	valBytes, ok := nodeFields[1].([]byte)
	if !ok {
		return fmt.Errorf("leaf node requires value byte slice")
	}
	if err := ma.AssembleKey().AssignString("PartialPath"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(partialPath); err != nil {
		return err
	}
	if err := ma.AssembleKey().AssignString("Value"); err != nil {
		return err
	}
	valUnionNodeMA, err := ma.AssembleValue().BeginMap(1)
	if err != nil {
		return err
	}
	if err := unpackValue(valUnionNodeMA, valBytes, codec); err != nil {
		return err
	}
	return valUnionNodeMA.Finish()
}

func unpackValue(ma ipld.MapAssembler, val []byte, codec uint64) error {
	switch codec {
	case cid.EthTxTrie:
		if err := ma.AssembleKey().AssignString(TX_VALUE.String()); err != nil {
			return err
		}
		return tx.DecodeBytes(ma.AssembleValue(), val)
	case cid.EthTxReceiptTrie:
		if err := ma.AssembleKey().AssignString(RCT_VALUE.String()); err != nil {
			return err
		}
		return rct.DecodeBytes(ma.AssembleValue(), val)
	case cid.EthStateTrie:
		if err := ma.AssembleKey().AssignString(STATE_VALUE.String()); err != nil {
			return err
		}
		return account.DecodeBytes(ma.AssembleValue(), val)
	case cid.EthStorageTrie:
		if err := ma.AssembleKey().AssignString(STORAGE_VALUE.String()); err != nil {
			return err
		}
		return ma.AssembleValue().AssignBytes(val)
	case logTrieMulticodec:
		if err := ma.AssembleKey().AssignString(LOG_VALUE.String()); err != nil {
			return err
		}
		return log.DecodeBytes(ma.AssembleValue(), val)
	default:
		return fmt.Errorf("unsupported multicodec type (%d) for eth TrieNode unmarshaller", codec)
	}
}

// decodeTwoMemberNode takes a two-member node, discerns its type and decodes its partial path before returning it
func decodeTwoMemberNode(i []interface{}) (NodeKind, []interface{}, error) {
	first := i[0].([]byte)
	decodedPartialPath := shared.CompactToHex(i[0].([]byte))
	decodedNode := []interface{}{
		decodedPartialPath,
		i[1],
	}
	switch first[0] / 16 {
	case '\x00', '\x01':
		return EXTENSION_NODE, decodedNode, nil
	case '\x02', '\x03':
		return LEAF_NODE, decodedNode, nil
	default:
		return UNKNOWN_NODE, nil, fmt.Errorf("unknown hex prefix")
	}
}
