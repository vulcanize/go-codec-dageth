package uncles_test

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	unc "github.com/vulcanize/go-codec-dageth/uncles"
)

var (
	uncle1 = &types.Header{
		ParentHash:  crypto.Keccak256Hash([]byte("uncle!")),
		Time:        1337,
		Coinbase:    common.HexToAddress("mockAddress"),
		Number:      new(big.Int).SetUint64(12244001),
		UncleHash:   common.Hash{},
		Root:        common.HexToHash("0x01"),
		TxHash:      common.HexToHash("0x02"),
		ReceiptHash: common.HexToHash("0x03"),
		Difficulty:  big.NewInt(5000000),
		Extra:       []byte{1, 2, 3},
		GasUsed:     1000000000,
		GasLimit:    1000000000,
		MixDigest:   crypto.Keccak256Hash([]byte("mixDigest")),
		Nonce:       types.BlockNonce{},
	}
	uncle2 = &types.Header{
		ParentHash:  crypto.Keccak256Hash([]byte("uncle, uncle!")),
		Time:        1338,
		Coinbase:    common.HexToAddress("mockAddress"),
		Number:      new(big.Int).SetUint64(12244002),
		Root:        common.HexToHash("0x01"),
		TxHash:      common.HexToHash("0x02"),
		ReceiptHash: common.HexToHash("0x03"),
		Difficulty:  big.NewInt(5000000),
		Extra:       []byte{1, 2, 3},
		GasUsed:     1000000000,
		GasLimit:    1000000000,
		MixDigest:   crypto.Keccak256Hash([]byte("mixDigest")),
		Nonce:       types.BlockNonce{},
	}
	uncles       = []*types.Header{uncle1, uncle2}
	unclesRLP, _ = rlp.EncodeToBytes(uncles)
	unclesNode   ipld.Node
)

/* IPLD Schema
type Uncles [Header]

type Header struct {
	ParentCID &Header
	UnclesCID &Uncles
	Coinbase Address
	StateRootCID &StateTrieNode
	TxRootCID &TxTrieNode
	RctRootCID &RctTrieNode
	Bloom Bloom
	Difficulty BigInt
	Number BigInt
	GasLimit Uint
	GasUsed Uint
	Time Time
	Extra Bytes
	MixDigest Hash
	Nonce Uint
}
*/

func TestUnclesCodec(t *testing.T) {
	testUnclesDecode(t)
	testUnclesNodeContents(t)
	testUnclesEncode(t)
}

func testUnclesDecode(t *testing.T) {
	unclesBuilder := dageth.Type.Uncles.NewBuilder()
	unclesReader := bytes.NewReader(unclesRLP)
	if err := unc.Decode(unclesBuilder, unclesReader); err != nil {
		t.Fatalf("unable to decode uncles into an IPLD node: %v", err)
	}
	unclesNode = unclesBuilder.Build()
}

func testUnclesNodeContents(t *testing.T) {
	unclesIT := unclesNode.ListIterator()
	for !unclesIT.Done() {
		i, uncleNode, err := unclesIT.Next()
		if err != nil {
			t.Fatalf("uncles iterator error: %v", err)
		}
		parentNode, err := uncleNode.LookupByString("ParentCID")
		if err != nil {
			t.Fatalf("uncles is missing ParentCID: %v", err)
		}
		parentLink, err := parentNode.AsLink()
		if err != nil {
			t.Fatalf("uncles ParentCID is not a link: %v", err)
		}
		parentCIDLink, ok := parentLink.(cidlink.Link)
		if !ok {
			t.Fatalf("uncles ParentCID is not a CID: %v", err)
		}
		parentMh := parentCIDLink.Hash()
		decodedParentMh, err := multihash.Decode(parentMh)
		if err != nil {
			t.Fatalf("uncles ParentCID could not be decoded into multihash: %v", err)
		}
		if !bytes.Equal(decodedParentMh.Digest, uncles[i].ParentHash.Bytes()) {
			t.Errorf("uncles parent hash (%x) does not match expected hash (%x)", decodedParentMh.Digest, uncles[i].ParentHash.Bytes())
		}

		unclesHashNode, err := uncleNode.LookupByString("UnclesCID")
		if err != nil {
			t.Fatalf("uncles is missing UnclesCID: %v", err)
		}
		unclesLink, err := unclesHashNode.AsLink()
		if err != nil {
			t.Fatalf("uncles UnclesCID is not a link: %v", err)
		}
		unclesCIDLink, ok := unclesLink.(cidlink.Link)
		if !ok {
			t.Fatalf("uncles UnclesCID is not a CID: %v", err)
		}
		unclesMh := unclesCIDLink.Hash()
		decodedUnclesMh, err := multihash.Decode(unclesMh)
		if err != nil {
			t.Fatalf("uncles UnclesCID could not be decoded into multihash: %v", err)
		}
		if !bytes.Equal(decodedParentMh.Digest, uncles[i].ParentHash.Bytes()) {
			t.Errorf("uncles uncles hash (%x) does not match expected hash (%x)", decodedUnclesMh.Digest, uncles[i].UncleHash.Bytes())
		}

		coinbaseNode, err := uncleNode.LookupByString("Coinbase")
		if err != nil {
			t.Fatalf("uncles is missing Coinbase: %v", err)
		}
		coinbaseBytes, err := coinbaseNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles Coinbase should be of type Bytes: %v", err)
		}
		if !bytes.Equal(coinbaseBytes, uncles[i].Coinbase.Bytes()) {
			t.Errorf("uncles coinbase address (%x) does not match expected address (%x)", coinbaseBytes, uncles[i].Coinbase.Bytes())
		}

		stateRootNode, err := uncleNode.LookupByString("StateRootCID")
		if err != nil {
			t.Fatalf("uncles is missing StateRootCID: %v", err)
		}
		stateRootLink, err := stateRootNode.AsLink()
		if err != nil {
			t.Fatalf("uncles StateRootCID is not a link: %v", err)
		}
		stateRootCIDLink, ok := stateRootLink.(cidlink.Link)
		if !ok {
			t.Fatalf("uncles StateRootCID is not a CID: %v", err)
		}
		stateRootMh := stateRootCIDLink.Hash()
		decodedStateRootMh, err := multihash.Decode(stateRootMh)
		if err != nil {
			t.Fatalf("uncles StateRootCID could not be decoded into multihash: %v", err)
		}
		if !bytes.Equal(decodedParentMh.Digest, uncles[i].ParentHash.Bytes()) {
			t.Errorf("uncles state root hash (%x) does not match expected hash (%x)", decodedStateRootMh.Digest, uncles[i].UncleHash.Bytes())
		}

		txRootNode, err := uncleNode.LookupByString("TxRootCID")
		if err != nil {
			t.Fatalf("uncles is missing TxRootCID: %v", err)
		}
		txRootLink, err := txRootNode.AsLink()
		if err != nil {
			t.Fatalf("uncles TxRootCID is not a link: %v", err)
		}
		txRootCIDLink, ok := txRootLink.(cidlink.Link)
		if !ok {
			t.Fatalf("uncles TxRootCID is not a CID: %v", err)
		}
		txRootMh := txRootCIDLink.Hash()
		decodedTxRootMh, err := multihash.Decode(txRootMh)
		if err != nil {
			t.Fatalf("uncles TxRootCID could not be decoded into multihash: %v", err)
		}
		if !bytes.Equal(decodedParentMh.Digest, uncles[i].ParentHash.Bytes()) {
			t.Errorf("uncles tx root hash (%x) does not match expected hash (%x)", decodedTxRootMh.Digest, uncles[i].UncleHash.Bytes())
		}

		rctRootNode, err := uncleNode.LookupByString("RctRootCID")
		if err != nil {
			t.Fatalf("uncles is missing RctRootCID: %v", err)
		}
		rctRootLink, err := rctRootNode.AsLink()
		if err != nil {
			t.Fatalf("uncles RctRootCID is not a link: %v", err)
		}
		rctRootCIDLink, ok := rctRootLink.(cidlink.Link)
		if !ok {
			t.Fatalf("uncles RctRootCID is not a CID: %v", err)
		}
		rctRootMh := rctRootCIDLink.Hash()
		decodedRctRootMh, err := multihash.Decode(rctRootMh)
		if err != nil {
			t.Fatalf("uncles RctRootCID could not be decoded into multihash: %v", err)
		}
		if !bytes.Equal(decodedParentMh.Digest, uncles[i].ParentHash.Bytes()) {
			t.Errorf("uncles rct root hash (%x) does not match expected hash (%x)", decodedRctRootMh.Digest, uncles[i].UncleHash.Bytes())
		}

		bloomNode, err := uncleNode.LookupByString("Bloom")
		if err != nil {
			t.Fatalf("uncles is missing Bloom: %v", err)
		}
		bloomBytes, err := bloomNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles Bloom should be of type Bytes: %v", err)
		}
		if !bytes.Equal(bloomBytes, uncles[i].Bloom.Bytes()) {
			t.Errorf("uncles bloom bytes (%x) does not match expected bytes (%x)", bloomBytes, uncles[i].Bloom.Bytes())
		}

		diffNode, err := uncleNode.LookupByString("Difficulty")
		if err != nil {
			t.Fatalf("uncles is missing Difficulty: %v", err)
		}
		diffNodeBytes, err := diffNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles Difficulty should be of type Bytes: %v", err)
		}
		if !bytes.Equal(diffNodeBytes, uncles[i].Difficulty.Bytes()) {
			t.Errorf("uncles difficulty (%x) does not match expected difficulty (%x)", diffNodeBytes, uncles[i].Difficulty.Bytes())
		}

		numberNode, err := uncleNode.LookupByString("Number")
		if err != nil {
			t.Fatalf("uncles is missing Number: %v", err)
		}
		numberBytes, err := numberNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles Number should be of type Bytes: %v", err)
		}
		if !bytes.Equal(numberBytes, uncles[i].Number.Bytes()) {
			t.Errorf("uncles number (%x) does not match expected number (%x)", numberBytes, uncles[i].Number.Bytes())
		}

		gasLimitNode, err := uncleNode.LookupByString("GasLimit")
		if err != nil {
			t.Fatalf("uncles is missing GasLimit: %v", err)
		}
		gasLimitBytes, err := gasLimitNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles GasLimit should be of type Bytes: %v", err)
		}
		gasLimitUint := binary.BigEndian.Uint64(gasLimitBytes)
		if gasLimitUint != uncles[i].GasLimit {
			t.Errorf("uncles gasLimit (%d) does not match expected gasLimit (%d)", gasLimitUint, uncles[i].GasLimit)
		}

		gasUsedNode, err := uncleNode.LookupByString("GasUsed")
		if err != nil {
			t.Fatalf("uncles is missing GasUsed: %v", err)
		}
		gasUsedBytes, err := gasUsedNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles GasUsed should be of type Bytes: %v", err)
		}
		gasUsedUint := binary.BigEndian.Uint64(gasUsedBytes)
		if gasUsedUint != uncles[i].GasUsed {
			t.Errorf("uncles gasUsed (%d) does not match expected gasUsed (%d)", gasUsedUint, uncles[i].GasUsed)
		}

		timeNode, err := uncleNode.LookupByString("Time")
		if err != nil {
			t.Fatalf("uncles is missing Time: %v", err)
		}
		timeBytes, err := timeNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles GasUsed should be of type Bytes: %v", err)
		}
		timeUint := binary.BigEndian.Uint64(timeBytes)
		if timeUint != uncles[i].Time {
			t.Errorf("uncles time (%d) does not match expected time (%d)", timeUint, uncles[i].Time)
		}

		extraNode, err := uncleNode.LookupByString("Extra")
		if err != nil {
			t.Fatalf("uncles is missing Extra: %v", err)
		}
		extraBytes, err := extraNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles Extra should be of type Byets: %v", err)
		}
		if !bytes.Equal(extraBytes, uncles[i].Extra) {
			t.Errorf("uncles extra bytes (%x) does not match expected bytes (%x)", extraBytes, uncles[i].Extra)
		}

		mixDigestNode, err := uncleNode.LookupByString("MixDigest")
		if err != nil {
			t.Fatalf("uncles is missing MixDigest: %v", err)
		}
		mixDigestBytes, err := mixDigestNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles Extra should be of type Byets: %v", err)
		}
		if !bytes.Equal(mixDigestBytes, uncles[i].MixDigest.Bytes()) {
			t.Errorf("uncles mixDigest bytes (%x) does not match expected bytes (%x)", mixDigestBytes, uncles[i].MixDigest.Bytes())
		}

		nonceNode, err := uncleNode.LookupByString("Nonce")
		if err != nil {
			t.Fatalf("uncles is missing Nonce: %v", err)
		}
		nonceBytes, err := nonceNode.AsBytes()
		if err != nil {
			t.Fatalf("uncles Extra should be of type Byets: %v", err)
		}
		nonce := binary.BigEndian.Uint64(nonceBytes)
		if nonce != uncles[i].Nonce.Uint64() {
			t.Errorf("uncles nonce (%d) does not match expected nonce (%d)", nonce, uncles[i].Nonce.Uint64())
		}
	}
}

func testUnclesEncode(t *testing.T) {
	unclesWriter := new(bytes.Buffer)
	if err := unc.Encode(unclesNode, unclesWriter); err != nil {
		t.Fatalf("unable to encode uncles into writer: %v", err)
	}
	encodedUnclesBytes := unclesWriter.Bytes()
	if !bytes.Equal(encodedUnclesBytes, unclesRLP) {
		t.Errorf("uncles encoding (%x) does not match the expected RLP encoding (%x)", encodedUnclesBytes, unclesRLP)
	}
}
