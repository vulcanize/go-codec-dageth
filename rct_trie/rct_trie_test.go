package rct_trie_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/rct_trie"
	"github.com/vulcanize/go-codec-dageth/shared"
	"github.com/vulcanize/go-codec-dageth/trie"
)

var (
	mockHash      = crypto.Keccak256([]byte{1, 2, 3, 4, 5})
	legacyReceipt = &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("hello"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("goodbye"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
		Type: types.LegacyTxType,
	}
	accessListReceipt = &types.Receipt{
		PostState:         mockHash,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{
				Address: common.BytesToAddress([]byte{0x11}),
				Topics:  []common.Hash{common.HexToHash("hello"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
			{
				Address: common.BytesToAddress([]byte{0x01, 0x11}),
				Topics:  []common.Hash{common.HexToHash("goodbye"), common.HexToHash("world")},
				Data:    []byte{0x01, 0x00, 0xff},
			},
		},
		Type: types.AccessListTxType,
	}
	mockLegacyReceiptLeafVal, _ = legacyReceipt.MarshalBinary()
	mockALReceiptLeafVal, _     = accessListReceipt.MarshalBinary()
	mockLeafParitalPath         = common.Hex2Bytes("3114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45")
	mockDecodedLeafPartialPath  = shared.CompactToHex(mockLeafParitalPath)
	mockLeafNodeLegacyReceipt   = []interface{}{
		mockLeafParitalPath,
		mockLegacyReceiptLeafVal,
	}
	mockLeafNodeALReceipt = []interface{}{
		mockLeafParitalPath,
		mockALReceiptLeafVal,
	}
	mockLeafNodeRLPLegacyReceipt, _ = rlp.EncodeToBytes(mockLeafNodeLegacyReceipt)
	mockLeafNodeRLPALReceipt, _     = rlp.EncodeToBytes(mockLeafNodeALReceipt)
	mockExtensionPartialPath        = common.Hex2Bytes("1114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45")
	mockDecodedExtensionPartialPath = shared.CompactToHex(mockExtensionPartialPath)
	mockExtensionHash               = crypto.Keccak256(mockLeafNodeRLPLegacyReceipt)
	mockExtensionNode               = []interface{}{
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
		mockLegacyReceiptLeafVal,
	}
	mockBranchNodeRLP, _ = rlp.EncodeToBytes(mockBranchNode)

	leafNodeLegacy, leafNodeAL, extensionNode, branchNode ipld.Node
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

type Topics [Hash]

type Log struct {
	Address Address
	Topics  Topics
	Data    Bytes
}

type Logs [Log]

type Receipt struct {
	TxType			  TxType
	// We could make Status an enum
	Status	          Uint   // nullable
	PostState		  Hash   // nullable
	CumulativeGasUsed Uint
	Bloom             Bloom
	Logs              Logs
}

type Receipts [Receipt]
*/

func TestReceiptTrieCodec(t *testing.T) {
	testReceiptTrieDecode(t)
	testReceiptTrieNodeContents(t)
	testReceiptTrieEncode(t)
}

func testReceiptTrieDecode(t *testing.T) {
	branchNodeBuilder := dageth.Type.TrieNode.NewBuilder()
	branchNodeReader := bytes.NewReader(mockBranchNodeRLP)
	if err := rct_trie.Decode(branchNodeBuilder, branchNodeReader); err != nil {
		t.Fatalf("unable to decode receipt trie branch node into an IPLD node: %v", err)
	}
	branchNode = branchNodeBuilder.Build()

	extensionNodeBuilder := dageth.Type.TrieNode.NewBuilder()
	extensionNodeReader := bytes.NewReader(mockExtensionNodeRLP)
	if err := rct_trie.Decode(extensionNodeBuilder, extensionNodeReader); err != nil {
		t.Fatalf("unable to decode receipt trie extension node into an IPLD node: %v", err)
	}
	extensionNode = extensionNodeBuilder.Build()

	leafNodeBuilderLegacy := dageth.Type.TrieNode.NewBuilder()
	leafNodeReaderLegacy := bytes.NewReader(mockLeafNodeRLPLegacyReceipt)
	if err := rct_trie.Decode(leafNodeBuilderLegacy, leafNodeReaderLegacy); err != nil {
		t.Fatalf("unable to decode receipt trie leaf node into an IPLD node: %v", err)
	}
	leafNodeLegacy = leafNodeBuilderLegacy.Build()

	leafNodeBuilderAL := dageth.Type.TrieNode.NewBuilder()
	leafNodeReaderAL := bytes.NewReader(mockLeafNodeRLPALReceipt)
	if err := rct_trie.Decode(leafNodeBuilderAL, leafNodeReaderAL); err != nil {
		t.Fatalf("unable to decode receipt trie leaf node into an IPLD node: %v", err)
	}
	leafNodeAL = leafNodeBuilderAL.Build()
}

func testReceiptTrieNodeContents(t *testing.T) {
	verifyBranchNodeContents(t)
	verifyExtensionNodeContents(t)
	verifyLegacyReceiptLeafNodeContents(t)
	verifyALReceiptLeafNodeContents(t)
}

func verifyBranchNodeContents(t *testing.T) {
	branch, err := branchNode.LookupByString(trie.BRANCH_NODE.String())
	if err != nil {
		t.Fatalf("receipt trie branch node missing enum key: %v", err)
	}
	nullChildren := []int{1, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 15}
	for _, i := range nullChildren {
		key := fmt.Sprintf("Child%s", strings.ToUpper(strconv.FormatInt(int64(i), 16)))
		childNode, err := branch.LookupByString(key)
		if err != nil {
			t.Fatalf("receipt trie branch node missing %s: %v", key, err)
		}
		if !childNode.IsNull() {
			t.Errorf("receipt trie branch node %s should be null", key)
		}
	}
	child0Node, err := branch.LookupByString("Child0")
	if err != nil {
		t.Fatalf("receipt trie branch node missing Child0: %v", err)
	}
	// Why do we have to look up the Union as if it is keyed representation?
	// It is kinded, we ought to be able to assert the node to a Link (call .AsLink on child0Node)
	child0LinkNode, err := child0Node.LookupByString("Link")
	if err != nil {
		t.Fatalf("receipt trie branch node Child0 should be of type Link: %v", err)
	}
	child0Link, err := child0LinkNode.AsLink()
	if err != nil {
		t.Fatalf("receipt trie branch node Child0 should be of type Link: %v", err)
	}
	child0CIDLink, ok := child0Link.(cidlink.Link)
	if !ok {
		t.Fatalf("receipt trie branch node Child0 should be a CID: %v", err)
	}
	child0Mh := child0CIDLink.Hash()
	decodedChild0Mh, err := multihash.Decode(child0Mh)
	if err != nil {
		t.Fatalf("could not decode branch node Child0 multihash: %v", err)
	}
	if !bytes.Equal(decodedChild0Mh.Digest, mockChild0) {
		t.Errorf("receipt trie branch node child0 hash (%x) does not match expected hash (%d)", decodedChild0Mh.Digest, mockChild0)
	}

	child5Node, err := branch.LookupByString("Child5")
	if err != nil {
		t.Fatalf("receipt trie branch node missing Child5: %v", err)
	}
	child5LinkNode, err := child5Node.LookupByString("Link")
	if err != nil {
		t.Fatalf("receipt trie branch node Child5 should be of type Link: %v", err)
	}
	child5Link, err := child5LinkNode.AsLink()
	if err != nil {
		t.Fatalf("receipt trie branch node Child5 should be of type Link: %v", err)
	}
	child5CIDLink, ok := child5Link.(cidlink.Link)
	if !ok {
		t.Fatalf("receipt trie branch node Child5 should be a CID: %v", err)
	}
	child5Mh := child5CIDLink.Hash()
	decodedChild5Mh, err := multihash.Decode(child5Mh)
	if err != nil {
		t.Fatalf("could not decode branch node Child5 multihash: %v", err)
	}
	if !bytes.Equal(decodedChild5Mh.Digest, mockChild5) {
		t.Errorf("receipt trie branch node child5 hash (%x) does not match expected hash (%d)", decodedChild5Mh.Digest, mockChild0)
	}

	childENode, err := branch.LookupByString("ChildE")
	if err != nil {
		t.Fatalf("receipt trie branch node missing ChildE: %v", err)
	}
	childELinkNode, err := childENode.LookupByString("Link")
	if err != nil {
		t.Fatalf("receipt trie branch node ChildE should be of type Link: %v", err)
	}
	childELink, err := childELinkNode.AsLink()
	if err != nil {
		t.Fatalf("receipt trie branch node ChildE should be of type Link: %v", err)
	}
	childECIDLink, ok := childELink.(cidlink.Link)
	if !ok {
		t.Fatalf("receipt trie branch node ChildE should be a CID: %v", err)
	}
	childEMh := childECIDLink.Hash()
	decodedChildEMh, err := multihash.Decode(childEMh)
	if err != nil {
		t.Fatalf("could not decode branch node ChildE multihash: %v", err)
	}
	if !bytes.Equal(decodedChildEMh.Digest, mockChildE) {
		t.Errorf("receipt trie branch node childE hash (%x) does not match expected hash (%d)", decodedChildEMh.Digest, mockChild0)
	}

	valEnumNode, err := branch.LookupByString("Value")
	if err != nil {
		t.Fatalf("receipt trie branch node missing Value: %v", err)
	}
	verifyLegacyReceiptLeafValue(valEnumNode, t)
}

func verifyExtensionNodeContents(t *testing.T) {
	ext, err := extensionNode.LookupByString(trie.EXTENSION_NODE.String())
	if err != nil {
		t.Fatalf("receipt trie extension node missing enum key: %v", err)
	}

	extPathNode, err := ext.LookupByString("PartialPath")
	if err != nil {
		t.Fatalf("receipt trie extension node missing PartialPath: %v", err)
	}
	extPathBytes, err := extPathNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt trie extension node PartialPath should be of type Bytes: %v", err)
	}
	if !bytes.Equal(extPathBytes, mockDecodedExtensionPartialPath) {
		t.Errorf("receipt trie extension node partial path (%x) does not match expected partial path (%x)", extPathBytes, mockExtensionPartialPath)
	}

	childNode, err := ext.LookupByString("Child")
	if err != nil {
		t.Fatalf("receipt trie extension node missing Child: %v", err)
	}
	childLink, err := childNode.AsLink()
	if err != nil {
		t.Fatalf("receipt trie extension node Child should be of kind Link: %v", err)
	}
	childCIDLink, ok := childLink.(cidlink.Link)
	if !ok {
		t.Fatalf("receipt trie extension node Child is not a CID: %v", err)
	}
	childMh := childCIDLink.Hash()
	decodedChildMh, err := multihash.Decode(childMh)
	if err != nil {
		t.Fatalf("receipt trie extension node Child could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedChildMh.Digest, mockExtensionHash) {
		t.Errorf("receipt trie extension node child hash (%x) does not match expected hash (%x)", decodedChildMh.Digest, mockExtensionHash)
	}
}

func verifyALReceiptLeafNodeContents(t *testing.T) {
	leaf, err := leafNodeAL.LookupByString(trie.LEAF_NODE.String())
	if err != nil {
		t.Fatalf("unable to resolve TrieNode union to a leaf: %v", err)
	}

	leafPathNode, err := leaf.LookupByString("PartialPath")
	if err != nil {
		t.Fatalf("receipt trie leaf node missing PartialPath: %v", err)
	}
	leafPathBytes, err := leafPathNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt trie leaf node PartialPath should be of type Bytes: %v", err)
	}

	if !bytes.Equal(leafPathBytes, mockDecodedLeafPartialPath) {
		t.Errorf("receipt trie leaf node partial path (%x) does not match expected partial path (%x)", leafPathBytes, mockDecodedLeafPartialPath)
	}

	valEnumNode, err := leaf.LookupByString("Value")
	if err != nil {
		t.Fatalf("receipt trie leaf node missing Value: %v", err)
	}
	verifyALReceiptLeafValue(valEnumNode, t)
}

func verifyLegacyReceiptLeafNodeContents(t *testing.T) {
	leaf, err := leafNodeLegacy.LookupByString(trie.LEAF_NODE.String())
	if err != nil {
		t.Fatalf("unable to resolve TrieNode union to a leaf: %v", err)
	}

	leafPathNode, err := leaf.LookupByString("PartialPath")
	if err != nil {
		t.Fatalf("receipt trie leaf node missing PartialPath: %v", err)
	}
	leafPathBytes, err := leafPathNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt trie leaf node PartialPath should be of type Bytes: %v", err)
	}

	if !bytes.Equal(leafPathBytes, mockDecodedLeafPartialPath) {
		t.Errorf("receipt trie leaf node partial path (%x) does not match expected partial path (%x)", leafPathBytes, mockDecodedLeafPartialPath)
	}

	valEnumNode, err := leaf.LookupByString("Value")
	if err != nil {
		t.Fatalf("receipt trie leaf node missing Value: %v", err)
	}
	verifyLegacyReceiptLeafValue(valEnumNode, t)
}

func verifyALReceiptLeafValue(valEnumNode ipld.Node, t *testing.T) {
	rctNode, err := valEnumNode.LookupByString(trie.RCT_VALUE.String())
	if err != nil {
		t.Fatalf("unable to resolve Value union to a receipt: %v", err)
	}

	verifySharedContent(t, rctNode, accessListReceipt)
	statusNode, err := rctNode.LookupByString("Status")
	if err != nil {
		t.Fatalf("receipt is missing Status: %v", err)
	}
	if !statusNode.IsNull() {
		t.Fatalf("receipt Status should be null")
	}

	postStateNode, err := rctNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState: %v", err)
	}
	if postStateNode.IsNull() {
		t.Errorf("receipt PostState should not be null")
	}
	postStateBy, err := postStateNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt PostState should be of type Bytes: %v", err)
	}
	if !bytes.Equal(postStateBy, accessListReceipt.PostState) {
		t.Errorf("receipt post state (%d) does not match expected post state (%d)", postStateBy, accessListReceipt.PostState)
	}
}

func verifyLegacyReceiptLeafValue(valEnumNode ipld.Node, t *testing.T) {
	rctNode, err := valEnumNode.LookupByString(trie.RCT_VALUE.String())
	if err != nil {
		t.Fatalf("unable to resolve Value union to a receipt: %v", err)
	}

	verifySharedContent(t, rctNode, legacyReceipt)
	statusNode, err := rctNode.LookupByString("Status")
	if err != nil {
		t.Fatalf("receipt is missing Status: %v", err)
	}
	if statusNode.IsNull() {
		t.Fatalf("receipt Status should not be null")
	}
	statusBy, err := statusNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt Status should be of type Bytes: %v", err)
	}
	status := binary.BigEndian.Uint64(statusBy)
	if status != legacyReceipt.Status {
		t.Errorf("receipt status (%d) does not match expected status (%d)", status, legacyReceipt.Status)
	}

	postStateNode, err := rctNode.LookupByString("PostState")
	if err != nil {
		t.Fatalf("receipt is missing PostState: %v", err)
	}
	if !postStateNode.IsNull() {
		t.Errorf("receipt PostState should be null")
	}
}

func verifySharedContent(t *testing.T, rctNode ipld.Node, rct *types.Receipt) {
	typeNode, err := rctNode.LookupByString("TxType")
	if err != nil {
		t.Fatalf("receipt is missing TxType: %v", err)
	}
	typeBy, err := typeNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt TxType should be of type Bytes: %v", err)
	}
	if len(typeBy) != 1 {
		t.Fatalf("receipt TxType should be a single byte")
	}
	if typeBy[0] != rct.Type {
		t.Errorf("receipt tx type (%d) does not match expected tx type (%d)", typeBy[0], rct.Type)
	}

	cguNode, err := rctNode.LookupByString("CumulativeGasUsed")
	if err != nil {
		t.Fatalf("receipt is missing CumulativeGasUsed: %v", err)
	}
	cguBy, err := cguNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt CumulativeGasUsed should be of type Bytes: %v", err)
	}
	cgu := binary.BigEndian.Uint64(cguBy)
	if cgu != rct.CumulativeGasUsed {
		t.Errorf("receipt cumulative gas used (%d) does not match expected cumulative gas used (%d)", cgu, rct.CumulativeGasUsed)
	}

	bloomNode, err := rctNode.LookupByString("Bloom")
	if err != nil {
		t.Fatalf("receipt is missing Bloom: %v", err)
	}
	bloomBy, err := bloomNode.AsBytes()
	if err != nil {
		t.Fatalf("receipt Bloom should be of type Bytes: %v", err)
	}
	if !bytes.Equal(bloomBy, rct.Bloom.Bytes()) {
		t.Errorf("receipt bloom (%x) does not match expected bloom (%x)", bloomBy, rct.Bloom.Bytes())
	}

	logsNode, err := rctNode.LookupByString("Logs")
	if err != nil {
		t.Fatalf("receipt is missing Logs: %v", err)
	}
	if logsNode.Length() != int64(len(rct.Logs)) {
		t.Fatalf("receipt should have %d logs", len(rct.Logs))
	}
	logsLI := logsNode.ListIterator()
	for !logsLI.Done() {
		i, logNode, err := logsLI.Next()
		if err != nil {
			t.Fatalf("receipt log iterator error: %v", err)
		}
		currentLog := rct.Logs[i]
		addrNode, err := logNode.LookupByString("Address")
		if err != nil {
			t.Fatalf("receipt log is missing Address: %v", err)
		}
		addrBy, err := addrNode.AsBytes()
		if err != nil {
			t.Fatalf("receipt log Address should be of type Bytes: %v", err)
		}
		if !bytes.Equal(addrBy, currentLog.Address.Bytes()) {
			t.Errorf("receipt log address (%x) does not match expected address (%x)", addrBy, currentLog.Address.Bytes())
		}
		dataNode, err := logNode.LookupByString("Data")
		if err != nil {
			t.Fatalf("receipt log is missing Data: %v", err)
		}
		data, err := dataNode.AsBytes()
		if err != nil {
			t.Fatalf("receipt log Data should be of type Bytes: %v", err)
		}
		if !bytes.Equal(data, currentLog.Data) {
			t.Errorf("receipt log data (%x) does not match expected data (%x)", data, currentLog.Data)
		}
		topicsNode, err := logNode.LookupByString("Topics")
		if err != nil {
			t.Fatalf("receipt log is missing Topics: %v", err)
		}
		if topicsNode.Length() != 2 {
			t.Fatal("receipt log should have two topics")
		}
		topicsLI := topicsNode.ListIterator()
		for !topicsLI.Done() {
			j, topicNode, err := topicsLI.Next()
			if err != nil {
				t.Fatalf("receipt log topic iterator error: %v", err)
			}
			currentTopic := currentLog.Topics[j].Bytes()
			topicBy, err := topicNode.AsBytes()
			if err != nil {
				t.Fatalf("receipt log Topic should be of type Bytes: %v", err)
			}
			if !bytes.Equal(topicBy, currentTopic) {
				t.Errorf("receipt log topic%d bytes (%x) does not match expected bytes (%x)", j, topicBy, currentTopic)
			}
		}
	}
}

func testReceiptTrieEncode(t *testing.T) {
	branchWriter := new(bytes.Buffer)
	if err := rct_trie.Encode(branchNode, branchWriter); err != nil {
		t.Fatalf("unable to encode receipt trie branch node into writer: %v", err)
	}
	encodedBranchBytes := branchWriter.Bytes()
	if !bytes.Equal(encodedBranchBytes, mockBranchNodeRLP) {
		t.Errorf("receipt trie branch node encoding (%x) does not match the expected consensus encoding (%x)", encodedBranchBytes, mockBranchNodeRLP)
	}

	extensionWriter := new(bytes.Buffer)
	if err := rct_trie.Encode(extensionNode, extensionWriter); err != nil {
		t.Fatalf("unable to encode receipt trie extension node into writer: %v", err)
	}
	encodedExtensionBytes := extensionWriter.Bytes()
	if !bytes.Equal(encodedExtensionBytes, mockExtensionNodeRLP) {
		t.Errorf("receipt trie extension node encoding (%x) does not match the expected consensus encoding (%x)", encodedExtensionBytes, mockExtensionNodeRLP)
	}

	leafWriterLegacyReceipt := new(bytes.Buffer)
	if err := rct_trie.Encode(leafNodeLegacy, leafWriterLegacyReceipt); err != nil {
		t.Fatalf("unable to encode receipt trie leaf node into writer: %v", err)
	}
	encodedLeafBytesLegacyReceipt := leafWriterLegacyReceipt.Bytes()
	if !bytes.Equal(encodedLeafBytesLegacyReceipt, mockLeafNodeRLPLegacyReceipt) {
		t.Errorf("receipt trie leaf node encoding (%x) does not match the expected consenus encoding (%x)", encodedLeafBytesLegacyReceipt, mockLeafNodeRLPLegacyReceipt)
	}

	leafWriterALReceipt := new(bytes.Buffer)
	if err := rct_trie.Encode(leafNodeAL, leafWriterALReceipt); err != nil {
		t.Fatalf("unable to encode receipt trie leaf node into writer: %v", err)
	}
	encodedLeafBytesALReceipt := leafWriterALReceipt.Bytes()
	if !bytes.Equal(encodedLeafBytesALReceipt, mockLeafNodeRLPALReceipt) {
		t.Errorf("receipt trie leaf node encoding (%x) does not match the expected consenus encoding (%x)", encodedLeafBytesALReceipt, mockLeafNodeRLPALReceipt)
	}
}
