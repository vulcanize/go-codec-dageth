package storage_trie_test

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/storage_trie"
	"github.com/vulcanize/go-codec-dageth/trie"
)

var (
	mockLeafVal, _      = rlp.EncodeToBytes([]byte{1, 2, 3, 4, 5})
	mockLeafParitalPath = common.Hex2Bytes("3114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45")
	mockLeafNode        = []interface{}{
		mockLeafParitalPath,
		mockLeafVal,
	}
	mockLeafNodeRLP, _       = rlp.EncodeToBytes(mockLeafNode)
	mockExtensionPartialPath = common.Hex2Bytes("1114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45")
	mockExtensionHash        = crypto.Keccak256(mockLeafNodeRLP)
	mockExtensionNode        = []interface{}{
		mockExtensionPartialPath,
		mockExtensionHash,
	}
	mockExtensionNodeRLP, _ = rlp.EncodeToBytes(mockExtensionNode)
	mockChild0              = crypto.Keccak256([]byte{0})
	mockChild5              = crypto.Keccak256([]byte{5})
	mockChildE              = crypto.Keccak256([]byte{14})
	mockBranchNode          = []interface{}{
		mockChild0,
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		mockChild5,
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		[]byte{},
		mockChildE,
		[]byte{},
		mockLeafVal,
	}
	mockBranchNodeRLP, _ = rlp.EncodeToBytes(mockBranchNode)

	leafNode, extensionNode, branchNode ipld.Node
)

/* IPLD Schemas
type TrieNode union {
	| TrieBranchNode "branch"
	| TrieExtensionNode "extension"
	| TrieLeafNode "leaf"
} representation keyed

type TrieBranchNode struct {
	Child0 nullable Child
	Child1 nullable Child
	Child2 nullable Child
	Child3 nullable Child
	Child4 nullable Child
	Child5 nullable Child
	Child6 nullable Child
	Child7 nullable Child
	Child8 nullable Child
	Child9 nullable Child
	ChildA nullable Child
	ChildB nullable Child
	ChildC nullable Child
	ChildD nullable Child
	ChildE nullable Child
	ChildF nullable Child
	Value  nullable Value
}

type Value union {
	| Transaction "tx"
	| Receipt "rct"
	| Account "state"
	| Bytes "storage"
} representation keyed

type Child union {
	| Link &TrieNode
	| TrieNode TrieNode
} representation kinded

type TrieExtensionNode struct {
	PartialPath Bytes
	Child Child
}

type TrieLeafNode struct {
	PartialPath Bytes
	Value       Value
}
*/

func TestStorageTrieCodec(t *testing.T) {
	testStorageTrieDecode(t)
	testStorageTrieNodeContents(t)
	testStorageTrieEncode(t)
}

func testStorageTrieDecode(t *testing.T) {
	branchNodeBuilder := dageth.Type.TrieNode.NewBuilder()
	branchNodeReader := bytes.NewReader(mockBranchNodeRLP)
	if err := storage_trie.Decode(branchNodeBuilder, branchNodeReader); err != nil {
		t.Fatalf("unable to decode storage trie branch node into an IPLD node: %v", err)
	}
	branchNode = branchNodeBuilder.Build()

	extensionNodeBuilder := dageth.Type.TrieNode.NewBuilder()
	extensionNodeReader := bytes.NewReader(mockExtensionNodeRLP)
	if err := storage_trie.Decode(extensionNodeBuilder, extensionNodeReader); err != nil {
		t.Fatalf("unable to decode storage trie extension node into an IPLD node: %v", err)
	}
	extensionNode = extensionNodeBuilder.Build()

	leafNodeBuilder := dageth.Type.TrieNode.NewBuilder()
	leafNodeReader := bytes.NewReader(mockLeafNodeRLP)
	if err := storage_trie.Decode(leafNodeBuilder, leafNodeReader); err != nil {
		t.Fatalf("unable to decode storage trie leaf node into an IPLD node: %v", err)
	}
	leafNode = leafNodeBuilder.Build()
}

func testStorageTrieNodeContents(t *testing.T) {
	verifyBranchNodeContents(t)
	verifyExtensionNodeContents(t)
	verifyLeafNodeContents(t)
}

func verifyBranchNodeContents(t *testing.T) {
	branch, err := branchNode.LookupByString(trie.BRANCH_NODE.String())
	if err != nil {
		t.Fatalf("storage trie branch node missing enum key: %v", err)
	}
	nullChildren := []int{1, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 15}
	for _, i := range nullChildren {
		key := fmt.Sprintf("Child%s", strings.ToUpper(strconv.FormatInt(int64(i), 16)))
		childNode, err := branch.LookupByString(key)
		if err != nil {
			t.Fatalf("storage trie branch node missing %s: %v", key, err)
		}
		if !childNode.IsNull() {
			t.Errorf("storage trie branch node %s should be null", key)
		}
	}
	child0Node, err := branch.LookupByString("Child0")
	if err != nil {
		t.Fatalf("storage trie branch node missing Child0: %v", err)
	}
	child0Bytes, err := child0Node.AsBytes()
	if err != nil {
		t.Fatalf("storage trie branch node Child0 should be of type Bytes: %v", err)
	}
	if !bytes.Equal(child0Bytes, mockChild0) {
		t.Errorf("storage trie branch node child0 hash (%x) does not match expected hash (%d)", child0Bytes, mockChild0)
	}

	child5Node, err := branch.LookupByString("Child5")
	if err != nil {
		t.Fatalf("storage trie branch node missing Child5: %v", err)
	}
	child5Bytes, err := child5Node.AsBytes()
	if err != nil {
		t.Fatalf("storage trie branch node Child5 should be of type Bytes: %v", err)
	}
	if !bytes.Equal(child5Bytes, mockChild5) {
		t.Errorf("storage trie branch node child5 hash (%x) does not match expected hash (%d)", child5Bytes, mockChild5)
	}

	childENode, err := branch.LookupByString("ChildE")
	if err != nil {
		t.Fatalf("storage trie branch node missing ChildE: %v", err)
	}
	childEBytes, err := childENode.AsBytes()
	if err != nil {
		t.Fatalf("storage trie branch node ChildE should be of type Bytes: %v", err)
	}
	if !bytes.Equal(childEBytes, mockChildE) {
		t.Errorf("storage trie branch node childE hash (%x) does not match expected hash (%d)", childEBytes, mockChildE)
	}

	valEnumNode, err := branch.LookupByString("Value")
	if err != nil {
		t.Fatalf("storage trie branch node missing Value: %v", err)
	}
	valNode, err := valEnumNode.LookupByString("storage")
	if err != nil {
		t.Fatalf("unable to resolve Value union to storage bytes: %v", err)
	}
	valBytes, err := valNode.AsBytes()
	if err != nil {
		t.Fatalf("storage trie branch node Value should be of type Bytes")
	}
	if !bytes.Equal(valBytes, mockLeafVal) {
		t.Errorf("storage trie branch node value (%x) does not match expected value (%x)", valBytes, mockLeafVal)
	}
}

func verifyExtensionNodeContents(t *testing.T) {
	ext, err := extensionNode.LookupByString(trie.EXTENSION_NODE.String())
	if err != nil {
		t.Fatalf("storage trie extension node missing enum key: %v", err)
	}

	extPathNode, err := ext.LookupByString("PartialPath")
	if err != nil {
		t.Fatalf("storage trie extension node missing PartialPath: %v", err)
	}
	extPathBytes, err := extPathNode.AsBytes()
	if err != nil {
		t.Fatalf("storage trie extension node PartialPath should be of type Bytes: %v", err)
	}
	if !bytes.Equal(extPathBytes, mockExtensionPartialPath) {
		t.Errorf("storage trie extension node partial path (%x) does not match expected partial path (%x)", extPathBytes, mockExtensionPartialPath)
	}

	childNode, err := ext.LookupByString("Child")
	if err != nil {
		t.Fatalf("storage trie extension node missing Child: %v", err)
	}
	childLink, err := childNode.AsLink()
	if err != nil {
		t.Fatalf("storage trie extension node Child should be of kind Link: %v", err)
	}
	childCIDLink, ok := childLink.(cidlink.Link)
	if !ok {
		t.Fatalf("storage trie extension node Child is not a CID: %v", err)
	}
	childMh := childCIDLink.Hash()
	decodedChildMh, err := multihash.Decode(childMh)
	if err != nil {
		t.Fatalf("storage trie extension node Child could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedChildMh.Digest, mockExtensionHash) {
		t.Errorf("storage trie extension node child hash (%x) does not match expected hash (%x)", decodedChildMh.Digest, mockExtensionHash)
	}
}

func verifyLeafNodeContents(t *testing.T) {
	leaf, err := leafNode.LookupByString(trie.LEAF_NODE.String())
	if err != nil {
		t.Fatalf("unable to resolve TrieNode union to a leaf: %v", err)
	}

	leafPathNode, err := leaf.LookupByString("PartialPath")
	if err != nil {
		t.Fatalf("storage trie leaf node missing PartialPath: %v", err)
	}
	leafPathBytes, err := leafPathNode.AsBytes()
	if err != nil {
		t.Fatalf("storage trie leaf node PartialPath should be of type Bytes: %v", err)
	}
	if !bytes.Equal(leafPathBytes, mockLeafParitalPath) {
		t.Errorf("storage trie leaf node partial path (%x) does not match expected partial path (%x)", leafPathBytes, mockLeafParitalPath)
	}

	valEnumNode, err := leaf.LookupByString("Value")
	if err != nil {
		t.Fatalf("storage trie leaf node missing Value: %v", err)
	}
	valNode, err := valEnumNode.LookupByString("storage")
	if err != nil {
		t.Fatalf("unable to resolve Value union to a storage bytes: %v", err)
	}
	valBytes, err := valNode.AsBytes()
	if err != nil {
		t.Fatalf("storage trie leaf node Value should be of type Bytes")
	}
	if !bytes.Equal(valBytes, mockLeafVal) {
		t.Errorf("storage trie leaf node value (%x) does not match expected value (%x)", valBytes, mockLeafVal)
	}
}

func testStorageTrieEncode(t *testing.T) {
	branchWriter := new(bytes.Buffer)
	if err := storage_trie.Encode(branchNode, branchWriter); err != nil {
		t.Fatalf("unable to encode storage trie branch node into writer: %v", err)
	}
	encodedBranchBytes := branchWriter.Bytes()
	if !bytes.Equal(encodedBranchBytes, mockBranchNodeRLP) {
		t.Errorf("storage trie branch node encoding (%x) does not match the expected RLP encoding (%x)", encodedBranchBytes, mockBranchNodeRLP)
	}

	extensionWriter := new(bytes.Buffer)
	if err := storage_trie.Encode(extensionNode, extensionWriter); err != nil {
		t.Fatalf("unable to encode storage trie extension node into writer: %v", err)
	}
	encodedExtensionBytes := extensionWriter.Bytes()
	if !bytes.Equal(encodedExtensionBytes, mockExtensionNodeRLP) {
		t.Errorf("storage trie extension node encoding (%x) does not match the expected RLP encoding (%x)", encodedExtensionBytes, mockExtensionNodeRLP)
	}

	leafWriter := new(bytes.Buffer)
	if err := storage_trie.Encode(leafNode, leafWriter); err != nil {
		t.Fatalf("unable to encode storage trie leaf node into writer: %v", err)
	}
	encodedLeafBytes := leafWriter.Bytes()
	if !bytes.Equal(encodedLeafBytes, mockLeafNodeRLP) {
		t.Errorf("storage trie leaf node encoding (%x) does not match the expected RLP encoding (%x)", encodedLeafBytes, mockLeafNodeRLP)
	}
}
