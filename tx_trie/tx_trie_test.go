package tx_trie_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
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
	"github.com/vulcanize/go-codec-dageth/shared"
	"github.com/vulcanize/go-codec-dageth/trie"
	"github.com/vulcanize/go-codec-dageth/tx_trie"
)

var (
	testAddr        = common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b")
	testAddr2       = common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf1a")
	testStorageKey  = crypto.Keccak256Hash(testAddr.Bytes())
	testStorageKey2 = crypto.Keccak256Hash(testAddr2.Bytes())

	legacyTransaction, _ = types.NewTransaction(
		3,
		testAddr,
		big.NewInt(10),
		2000,
		big.NewInt(1),
		common.FromHex("5544"),
	).WithSignature(
		types.HomesteadSigner{},
		common.Hex2Bytes("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a301"),
	)

	accessListTransaction, _ = types.NewTx(&types.AccessListTx{
		ChainID:  big.NewInt(1),
		Nonce:    3,
		To:       &testAddr,
		Value:    big.NewInt(10),
		Gas:      25000,
		GasPrice: big.NewInt(1),
		Data:     common.FromHex("5544"),
		AccessList: types.AccessList{
			types.AccessTuple{
				Address: testAddr,
				StorageKeys: []common.Hash{
					testStorageKey,
					testStorageKey2,
				},
			},
			types.AccessTuple{
				Address:     testAddr2,
				StorageKeys: nil,
			},
		},
	}).WithSignature(
		types.NewEIP2930Signer(big.NewInt(1)),
		common.Hex2Bytes("c9519f4f2b30335884581971573fadf60c6204f59a911df35ee8a540456b266032f1e8e2c5dd761f9e4f88f41c8310aeaba26a8bfcdacfedfa12ec3862d3752101"),
	)
	mockLegacyTransactionLeafVal, _ = legacyTransaction.MarshalBinary()
	mockALTransactionLeafVal, _     = accessListTransaction.MarshalBinary()
	mockLeafParitalPath             = common.Hex2Bytes("3114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45")
	mockDecodedLeafPartialPath      = shared.CompactToHex(mockLeafParitalPath)
	mockLeafNodeLegacyTransaction   = []interface{}{
		mockLeafParitalPath,
		mockLegacyTransactionLeafVal,
	}
	mockLeafNodeALTransaction = []interface{}{
		mockLeafParitalPath,
		mockALTransactionLeafVal,
	}
	mockLeafNodeRLPLegacyTransaction, _ = rlp.EncodeToBytes(mockLeafNodeLegacyTransaction)
	mockLeafNodeRLPALTransaction, _     = rlp.EncodeToBytes(mockLeafNodeALTransaction)
	mockExtensionPartialPath            = common.Hex2Bytes("1114658a74d9cc9f7acf2c5cd696c3494d7c344d78bfec3add0d91ec4e8d1c45")
	mockDecodedExtensionPartialPath     = shared.CompactToHex(mockExtensionPartialPath)
	mockExtensionHash                   = crypto.Keccak256(mockLeafNodeRLPLegacyTransaction)
	mockExtensionNode                   = []interface{}{
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
		mockLegacyTransactionLeafVal,
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
	| Transaction "rct"
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

type StorageKeys [Hash]

type AccessElement struct {
	Address     Address
	StorageKeys StorageKeys
}

type AccessList [AccessElement]

type Transaction struct {
	TxType       TxType
	ChainID      nullable BigInt # null unless the transaction is an EIP-2930 transaction
	AccountNonce Uint
	GasPrice     BigInt
	GasLimit     Uint
	Recipient    nullable Address # null recipient means the tx is a contract creation
	Amount       BigInt
	Data         Bytes
	AccessList   nullable AccessList # null unless the transaction is an EIP-2930 transaction

	# Signature values
	V            BigInt
	R            BigInt
	S            BigInt
}
*/

func TestTransactionTrieCodec(t *testing.T) {
	testTransactionTrieDecode(t)
	testTransactionTrieNodeContents(t)
	testTransactionTrieEncode(t)
}

func testTransactionTrieDecode(t *testing.T) {
	branchNodeBuilder := dageth.Type.TrieNode.NewBuilder()
	branchNodeReader := bytes.NewReader(mockBranchNodeRLP)
	if err := tx_trie.Decode(branchNodeBuilder, branchNodeReader); err != nil {
		t.Fatalf("unable to decode transaction trie branch node into an IPLD node: %v", err)
	}
	branchNode = branchNodeBuilder.Build()

	extensionNodeBuilder := dageth.Type.TrieNode.NewBuilder()
	extensionNodeReader := bytes.NewReader(mockExtensionNodeRLP)
	if err := tx_trie.Decode(extensionNodeBuilder, extensionNodeReader); err != nil {
		t.Fatalf("unable to decode transaction trie extension node into an IPLD node: %v", err)
	}
	extensionNode = extensionNodeBuilder.Build()

	leafNodeBuilderLegacy := dageth.Type.TrieNode.NewBuilder()
	leafNodeReaderLegacy := bytes.NewReader(mockLeafNodeRLPLegacyTransaction)
	if err := tx_trie.Decode(leafNodeBuilderLegacy, leafNodeReaderLegacy); err != nil {
		t.Fatalf("unable to decode transaction trie leaf node into an IPLD node: %v", err)
	}
	leafNodeLegacy = leafNodeBuilderLegacy.Build()

	leafNodeBuilderAL := dageth.Type.TrieNode.NewBuilder()
	leafNodeReaderAL := bytes.NewReader(mockLeafNodeRLPALTransaction)
	if err := tx_trie.Decode(leafNodeBuilderAL, leafNodeReaderAL); err != nil {
		t.Fatalf("unable to decode transaction trie leaf node into an IPLD node: %v", err)
	}
	leafNodeAL = leafNodeBuilderAL.Build()
}

func testTransactionTrieNodeContents(t *testing.T) {
	verifyBranchNodeContents(t)
	verifyExtensionNodeContents(t)
	verifyLegacyTransactionLeafNodeContents(t)
	verifyALTransactionLeafNodeContents(t)
}

func verifyBranchNodeContents(t *testing.T) {
	branch, err := branchNode.LookupByString(trie.BRANCH_NODE.String())
	if err != nil {
		t.Fatalf("transaction trie branch node missing enum key: %v", err)
	}
	nullChildren := []int{1, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 15}
	for _, i := range nullChildren {
		key := fmt.Sprintf("Child%s", strings.ToUpper(strconv.FormatInt(int64(i), 16)))
		childNode, err := branch.LookupByString(key)
		if err != nil {
			t.Fatalf("transaction trie branch node missing %s: %v", key, err)
		}
		if !childNode.IsNull() {
			t.Errorf("transaction trie branch node %s should be null", key)
		}
	}
	child0Node, err := branch.LookupByString("Child0")
	if err != nil {
		t.Fatalf("transaction trie branch node missing Child0: %v", err)
	}
	// Why do we have to look up the Union as if it is keyed representation?
	// It is kinded, we ought to be able to assert the node to a Link (call .AsLink on child0Node)
	child0LinkNode, err := child0Node.LookupByString("Link")
	if err != nil {
		t.Fatalf("transaction trie branch node Child0 should be of type Link: %v", err)
	}
	child0Link, err := child0LinkNode.AsLink()
	if err != nil {
		t.Fatalf("transaction trie branch node Child0 should be of type Link: %v", err)
	}
	child0CIDLink, ok := child0Link.(cidlink.Link)
	if !ok {
		t.Fatalf("transaction trie branch node Child0 should be a CID: %v", err)
	}
	child0Mh := child0CIDLink.Hash()
	decodedChild0Mh, err := multihash.Decode(child0Mh)
	if err != nil {
		t.Fatalf("could not decode branch node Child0 multihash: %v", err)
	}
	if !bytes.Equal(decodedChild0Mh.Digest, mockChild0) {
		t.Errorf("transaction trie branch node child0 hash (%x) does not match expected hash (%d)", decodedChild0Mh.Digest, mockChild0)
	}

	child5Node, err := branch.LookupByString("Child5")
	if err != nil {
		t.Fatalf("transaction trie branch node missing Child5: %v", err)
	}
	child5LinkNode, err := child5Node.LookupByString("Link")
	if err != nil {
		t.Fatalf("transaction trie branch node Child5 should be of type Link: %v", err)
	}
	child5Link, err := child5LinkNode.AsLink()
	if err != nil {
		t.Fatalf("transaction trie branch node Child5 should be of type Link: %v", err)
	}
	child5CIDLink, ok := child5Link.(cidlink.Link)
	if !ok {
		t.Fatalf("transaction trie branch node Child5 should be a CID: %v", err)
	}
	child5Mh := child5CIDLink.Hash()
	decodedChild5Mh, err := multihash.Decode(child5Mh)
	if err != nil {
		t.Fatalf("could not decode branch node Child5 multihash: %v", err)
	}
	if !bytes.Equal(decodedChild5Mh.Digest, mockChild5) {
		t.Errorf("transaction trie branch node child5 hash (%x) does not match expected hash (%d)", decodedChild5Mh.Digest, mockChild0)
	}

	childENode, err := branch.LookupByString("ChildE")
	if err != nil {
		t.Fatalf("transaction trie branch node missing ChildE: %v", err)
	}
	childELinkNode, err := childENode.LookupByString("Link")
	if err != nil {
		t.Fatalf("transaction trie branch node ChildE should be of type Link: %v", err)
	}
	childELink, err := childELinkNode.AsLink()
	if err != nil {
		t.Fatalf("transaction trie branch node ChildE should be of type Link: %v", err)
	}
	childECIDLink, ok := childELink.(cidlink.Link)
	if !ok {
		t.Fatalf("transaction trie branch node ChildE should be a CID: %v", err)
	}
	childEMh := childECIDLink.Hash()
	decodedChildEMh, err := multihash.Decode(childEMh)
	if err != nil {
		t.Fatalf("could not decode branch node ChildE multihash: %v", err)
	}
	if !bytes.Equal(decodedChildEMh.Digest, mockChildE) {
		t.Errorf("transaction trie branch node childE hash (%x) does not match expected hash (%d)", decodedChildEMh.Digest, mockChild0)
	}

	valEnumNode, err := branch.LookupByString("Value")
	if err != nil {
		t.Fatalf("transaction trie branch node missing Value: %v", err)
	}
	verifyLegacyTransactionLeafValue(valEnumNode, t)
}

func verifyExtensionNodeContents(t *testing.T) {
	ext, err := extensionNode.LookupByString(trie.EXTENSION_NODE.String())
	if err != nil {
		t.Fatalf("transaction trie extension node missing enum key: %v", err)
	}

	extPathNode, err := ext.LookupByString("PartialPath")
	if err != nil {
		t.Fatalf("transaction trie extension node missing PartialPath: %v", err)
	}
	extPathBytes, err := extPathNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction trie extension node PartialPath should be of type Bytes: %v", err)
	}
	if !bytes.Equal(extPathBytes, mockDecodedExtensionPartialPath) {
		t.Errorf("transaction trie extension node partial path (%x) does not match expected partial path (%x)", extPathBytes, mockExtensionPartialPath)
	}

	childNode, err := ext.LookupByString("Child")
	if err != nil {
		t.Fatalf("transaction trie extension node missing Child: %v", err)
	}
	childLink, err := childNode.AsLink()
	if err != nil {
		t.Fatalf("transaction trie extension node Child should be of kind Link: %v", err)
	}
	childCIDLink, ok := childLink.(cidlink.Link)
	if !ok {
		t.Fatalf("transaction trie extension node Child is not a CID: %v", err)
	}
	childMh := childCIDLink.Hash()
	decodedChildMh, err := multihash.Decode(childMh)
	if err != nil {
		t.Fatalf("transaction trie extension node Child could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedChildMh.Digest, mockExtensionHash) {
		t.Errorf("transaction trie extension node child hash (%x) does not match expected hash (%x)", decodedChildMh.Digest, mockExtensionHash)
	}
}

func verifyALTransactionLeafNodeContents(t *testing.T) {
	leaf, err := leafNodeAL.LookupByString(trie.LEAF_NODE.String())
	if err != nil {
		t.Fatalf("unable to resolve TrieNode union to a leaf: %v", err)
	}

	leafPathNode, err := leaf.LookupByString("PartialPath")
	if err != nil {
		t.Fatalf("transaction trie leaf node missing PartialPath: %v", err)
	}
	leafPathBytes, err := leafPathNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction trie leaf node PartialPath should be of type Bytes: %v", err)
	}

	if !bytes.Equal(leafPathBytes, mockDecodedLeafPartialPath) {
		t.Errorf("transaction trie leaf node partial path (%x) does not match expected partial path (%x)", leafPathBytes, mockDecodedLeafPartialPath)
	}

	valEnumNode, err := leaf.LookupByString("Value")
	if err != nil {
		t.Fatalf("transaction trie leaf node missing Value: %v", err)
	}
	verifyALTransactionLeafValue(valEnumNode, t)
}

func verifyLegacyTransactionLeafNodeContents(t *testing.T) {
	leaf, err := leafNodeLegacy.LookupByString(trie.LEAF_NODE.String())
	if err != nil {
		t.Fatalf("unable to resolve TrieNode union to a leaf: %v", err)
	}

	leafPathNode, err := leaf.LookupByString("PartialPath")
	if err != nil {
		t.Fatalf("transaction trie leaf node missing PartialPath: %v", err)
	}
	leafPathBytes, err := leafPathNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction trie leaf node PartialPath should be of type Bytes: %v", err)
	}

	if !bytes.Equal(leafPathBytes, mockDecodedLeafPartialPath) {
		t.Errorf("transaction trie leaf node partial path (%x) does not match expected partial path (%x)", leafPathBytes, mockDecodedLeafPartialPath)
	}

	valEnumNode, err := leaf.LookupByString("Value")
	if err != nil {
		t.Fatalf("transaction trie leaf node missing Value: %v", err)
	}
	verifyLegacyTransactionLeafValue(valEnumNode, t)
}

func verifyALTransactionLeafValue(valEnumNode ipld.Node, t *testing.T) {
	txNode, err := valEnumNode.LookupByString(trie.TX_VALUE.String())
	if err != nil {
		t.Fatalf("unable to resolve Value union to a transaction: %v", err)
	}

	verifySharedContent(t, txNode, accessListTransaction)
	accessListNode, err := txNode.LookupByString("AccessList")
	if err != nil {
		t.Fatalf("transaction missing AccessList: %v", err)
	}
	if accessListNode.IsNull() {
		t.Fatalf("access list transaction AccessList should not be null")
	}
	if accessListNode.Length() != int64(len(accessListTransaction.AccessList())) {
		t.Fatalf("transaction access list should have %d elements", len(accessListTransaction.AccessList()))
	}
	accessListIT := accessListNode.ListIterator()
	for !accessListIT.Done() {
		i, accessListElementNode, err := accessListIT.Next()
		if err != nil {
			t.Fatalf("transaction access list iterator error: %v", err)
		}
		currentAccessListElement := accessListTransaction.AccessList()[i]
		addressNode, err := accessListElementNode.LookupByString("Address")
		if err != nil {
			t.Fatalf("transaction access list missing Address: %v", err)
		}
		addressBytes, err := addressNode.AsBytes()
		if err != nil {
			t.Fatalf("transaction access list Address should be of type Bytes: %v", err)
		}
		if !bytes.Equal(addressBytes, currentAccessListElement.Address.Bytes()) {
			t.Errorf("transaction access list address (%x) does not match expected address (%x)", addressBytes, currentAccessListElement.Address.Bytes())
		}

		storageKeysNode, err := accessListElementNode.LookupByString("StorageKeys")
		if err != nil {
			t.Fatalf("transaction access list missing StorageKeys: %v", err)
		}
		if storageKeysNode.Length() != int64(len(currentAccessListElement.StorageKeys)) {
			t.Fatalf("transaction access list storage keys should have %d entries", len(currentAccessListElement.StorageKeys))
		}
		storageKeyIT := storageKeysNode.ListIterator()
		for !storageKeyIT.Done() {
			j, storageKeyNode, err := storageKeyIT.Next()
			if err != nil {
				t.Fatalf("transaction access list storage keys iterator error: %v", err)
			}
			currentStorageKey := currentAccessListElement.StorageKeys[j]
			storageKeyBytes, err := storageKeyNode.AsBytes()
			if err != nil {
				t.Fatalf("transaction access list StorageKey should be of type Bytes: %v", err)
			}
			if !bytes.Equal(storageKeyBytes, currentStorageKey.Bytes()) {
				t.Errorf("transaction access list storage key (%x) does not match expected value (%x)", storageKeyBytes, currentStorageKey.Bytes())
			}
		}
	}

	idNode, err := txNode.LookupByString("ChainID")
	if err != nil {
		t.Fatalf("transaction is missing ChainID: %v", err)
	}
	if idNode.IsNull() {
		t.Fatalf("access list transaction ChainID should not be null")
	}
	idBytes, err := idNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction ChainID should be of type Bytes: %v", err)
	}
	if !bytes.Equal(idBytes, accessListTransaction.ChainId().Bytes()) {
		t.Errorf("transaction chain id (%x) does not match expected status (%x)", idBytes, accessListTransaction.ChainId().Bytes())
	}
}

func verifyLegacyTransactionLeafValue(valEnumNode ipld.Node, t *testing.T) {
	txNode, err := valEnumNode.LookupByString(trie.TX_VALUE.String())
	if err != nil {
		t.Fatalf("unable to resolve Value union to a transaction: %v", err)
	}

	verifySharedContent(t, txNode, legacyTransaction)
}

func verifySharedContent(t *testing.T, txNode ipld.Node, tx *types.Transaction) {
	v, r, s := tx.RawSignatureValues()
	vNode, err := txNode.LookupByString("V")
	if err != nil {
		t.Fatalf("transaction missing V: %v", err)
	}
	vBytes, err := vNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction V should be of type Bytes: %v", err)
	}
	if !bytes.Equal(vBytes, v.Bytes()) {
		t.Errorf("transaction v bytes (%x) does not match expected bytes (%x)", vBytes, v.Bytes())
	}

	rNode, err := txNode.LookupByString("R")
	if err != nil {
		t.Fatalf("transaction missing R: %v", err)
	}
	rBytes, err := rNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction R should be of type Bytes: %v", err)
	}
	if !bytes.Equal(rBytes, r.Bytes()) {
		t.Errorf("transaction r bytes (%x) does not match expected bytes (%x)", rBytes, r.Bytes())
	}

	sNode, err := txNode.LookupByString("S")
	if err != nil {
		t.Fatalf("transaction missing S: %v", err)
	}
	sBytes, err := sNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction S should be of type Bytes: %v", err)
	}
	if !bytes.Equal(sBytes, s.Bytes()) {
		t.Errorf("transaction s bytes (%x) does not match expected bytes (%x)", sBytes, s.Bytes())
	}

	dataNode, err := txNode.LookupByString("Data")
	if err != nil {
		t.Fatalf("transaction missing Data: %v", err)
	}
	dataBytes, err := dataNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Data should be of type Bytes: %v", err)
	}
	if !bytes.Equal(dataBytes, tx.Data()) {
		t.Errorf("transaction data bytes (%x) does not match expected bytes (%x)", dataBytes, tx.Data())
	}

	amountNode, err := txNode.LookupByString("Amount")
	if err != nil {
		t.Fatalf("transaction missing Amount: %v", err)
	}
	amountBytes, err := amountNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Amount should be of type Bytes: %v", err)
	}
	if !bytes.Equal(amountBytes, tx.Value().Bytes()) {
		t.Errorf("transaction amount (%x) does not match expected amount (%x)", amountBytes, tx.Value().Bytes())
	}

	recipientNode, err := txNode.LookupByString("Recipient")
	if err != nil {
		t.Fatalf("transaction missing Recipient: %v", err)
	}
	recipientBytes, err := recipientNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Recipient should be of type Bytes: %v", err)
	}
	if !bytes.Equal(recipientBytes, tx.To().Bytes()) {
		t.Errorf("transaction recipient (%x) does not match expected recipient (%x)", recipientBytes, tx.To().Bytes())
	}

	gasLimitNode, err := txNode.LookupByString("GasLimit")
	if err != nil {
		t.Fatalf("transaction missing GasLimit: %v", err)
	}
	gasLimitBytes, err := gasLimitNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasLimit should be of type Bytes: %v", err)
	}
	gas := binary.BigEndian.Uint64(gasLimitBytes)
	if gas != tx.Gas() {
		t.Errorf("transaction gas limit (%d) does not match expected gas limit (%d)", gas, tx.Gas())
	}

	gasPriceNode, err := txNode.LookupByString("GasPrice")
	if err != nil {
		t.Fatalf("transaction missing GasPrice: %v", err)
	}
	gasPriceBytes, err := gasPriceNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction GasPrice should be of type Bytes: %v", err)
	}
	if !bytes.Equal(gasPriceBytes, tx.GasPrice().Bytes()) {
		t.Errorf("transaction gas price (%x) does not match expected gas price (%x)", gasPriceBytes, tx.GasPrice().Bytes())
	}

	nonceNode, err := txNode.LookupByString("AccountNonce")
	if err != nil {
		t.Fatalf("transaction missing AccountNonce: %v", err)
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction Nonce should be of type Bytes: %v", err)
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	if nonce != tx.Nonce() {
		t.Errorf("transaction nonce (%d) does not match expected nonce (%d)", nonce, tx.Nonce())
	}

	typeNode, err := txNode.LookupByString("TxType")
	if err != nil {
		t.Fatalf("transaction missing TxType: %v", err)
	}
	typeBy, err := typeNode.AsBytes()
	if err != nil {
		t.Fatalf("transaction TxType should be of type Bytes: %v", err)
	}
	if len(typeBy) != 1 {
		t.Fatalf("transaction TxType should be a single byte")
	}
	if typeBy[0] != tx.Type() {
		t.Errorf("transaction tx type (%d) does not match expected tx type (%d)", typeBy[0], tx.Type())
	}
}

func testTransactionTrieEncode(t *testing.T) {
	branchWriter := new(bytes.Buffer)
	if err := tx_trie.Encode(branchNode, branchWriter); err != nil {
		t.Fatalf("unable to encode transaction trie branch node into writer: %v", err)
	}
	encodedBranchBytes := branchWriter.Bytes()
	if !bytes.Equal(encodedBranchBytes, mockBranchNodeRLP) {
		t.Errorf("transaction trie branch node encoding (%x) does not match the expected consensus encoding (%x)", encodedBranchBytes, mockBranchNodeRLP)
	}

	extensionWriter := new(bytes.Buffer)
	if err := tx_trie.Encode(extensionNode, extensionWriter); err != nil {
		t.Fatalf("unable to encode transaction trie extension node into writer: %v", err)
	}
	encodedExtensionBytes := extensionWriter.Bytes()
	if !bytes.Equal(encodedExtensionBytes, mockExtensionNodeRLP) {
		t.Errorf("transaction trie extension node encoding (%x) does not match the expected consensus encoding (%x)", encodedExtensionBytes, mockExtensionNodeRLP)
	}

	leafWriterLegacyTransaction := new(bytes.Buffer)
	if err := tx_trie.Encode(leafNodeLegacy, leafWriterLegacyTransaction); err != nil {
		t.Fatalf("unable to encode transaction trie leaf node into writer: %v", err)
	}
	encodedLeafBytesLegacyTransaction := leafWriterLegacyTransaction.Bytes()
	if !bytes.Equal(encodedLeafBytesLegacyTransaction, mockLeafNodeRLPLegacyTransaction) {
		t.Errorf("transaction trie leaf node encoding (%x) does not match the expected consenus encoding (%x)", encodedLeafBytesLegacyTransaction, mockLeafNodeRLPLegacyTransaction)
	}

	leafWriterALTransaction := new(bytes.Buffer)
	if err := tx_trie.Encode(leafNodeAL, leafWriterALTransaction); err != nil {
		t.Fatalf("unable to encode transaction trie leaf node into writer: %v", err)
	}
	encodedLeafBytesALTransaction := leafWriterALTransaction.Bytes()
	if !bytes.Equal(encodedLeafBytesALTransaction, mockLeafNodeRLPALTransaction) {
		t.Errorf("transaction trie leaf node encoding (%x) does not match the expected consenus encoding (%x)", encodedLeafBytesALTransaction, mockLeafNodeRLPALTransaction)
	}
}
