package header_test

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/header"
)

var (
	gethHeader *types.Header
	headerRLP  []byte
	headerNode ipld.Node
)

func TestHeaderCodec(t *testing.T) {
	block, _, err := loadBlockFromRLPFile("./block1_rlp")
	if err != nil {
		t.Fatal(err)
	}
	gethHeader = block.Header()
	headerRLP, err = rlp.EncodeToBytes(gethHeader)
	if err != nil {
		t.Fatal(err)
	}
	testHeaderDecode(t)
	testHeaderNodeContents(t)
	testHeaderEncode(t)
}

func testHeaderDecode(t *testing.T) {
	headerBuilder := dageth.Type.Header.NewBuilder()
	headerReader := bytes.NewReader(headerRLP)
	if err := header.Decode(headerBuilder, headerReader); err != nil {
		t.Fatalf("unable to decode header into an IPLD node: %v", err)
	}
	headerNode = headerBuilder.Build()
}

func testHeaderNodeContents(t *testing.T) {
	parentNode, err := headerNode.LookupByString("ParentCID")
	if err != nil {
		t.Fatalf("header is missing ParentCID: %v", err)
	}
	parentLink, err := parentNode.AsLink()
	if err != nil {
		t.Fatalf("header ParentCID is not a link: %v", err)
	}
	parentCIDLink, ok := parentLink.(cidlink.Link)
	if !ok {
		t.Fatalf("header ParentCID is not a CID: %v", err)
	}
	parentMh := parentCIDLink.Hash()
	decodedParentMh, err := multihash.Decode(parentMh)
	if err != nil {
		t.Fatalf("header ParentCID could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedParentMh.Digest, gethHeader.ParentHash.Bytes()) {
		t.Errorf("header parent hash (%x) does not match expected hash (%x)", decodedParentMh.Digest, gethHeader.ParentHash.Bytes())
	}

	unclesNode, err := headerNode.LookupByString("UnclesCID")
	if err != nil {
		t.Fatalf("header is missing UnclesCID")
	}
	unclesLink, err := unclesNode.AsLink()
	if err != nil {
		t.Fatalf("header UnclesCID is not a link: %v", err)
	}
	unclesCIDLink, ok := unclesLink.(cidlink.Link)
	if !ok {
		t.Fatalf("header UnclesCID is not a CID: %v", err)
	}
	unclesMh := unclesCIDLink.Hash()
	decodedUnclesMh, err := multihash.Decode(unclesMh)
	if err != nil {
		t.Fatalf("header UnclesCID could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedParentMh.Digest, gethHeader.ParentHash.Bytes()) {
		t.Errorf("header uncles hash (%x) does not match expected hash (%x)", decodedUnclesMh.Digest, gethHeader.UncleHash.Bytes())
	}

	coinbaseNode, err := headerNode.LookupByString("Coinbase")
	if err != nil {
		t.Fatalf("header is missing Coinbase")
	}
	coinbaseBytes, err := coinbaseNode.AsBytes()
	if err != nil {
		t.Fatalf("header Coinbase should be of type Bytes: %v", err)
	}
	if !bytes.Equal(coinbaseBytes, gethHeader.Coinbase.Bytes()) {
		t.Errorf("header coinbase address (%x) does not match expected address (%x)", coinbaseBytes, gethHeader.Coinbase.Bytes())
	}

	stateRootNode, err := headerNode.LookupByString("StateRootCID")
	if err != nil {
		t.Fatalf("header is missing StateRootCID")
	}
	stateRootLink, err := stateRootNode.AsLink()
	if err != nil {
		t.Fatalf("header StateRootCID is not a link: %v", err)
	}
	stateRootCIDLink, ok := stateRootLink.(cidlink.Link)
	if !ok {
		t.Fatalf("header StateRootCID is not a CID: %v", err)
	}
	stateRootMh := stateRootCIDLink.Hash()
	decodedStateRootMh, err := multihash.Decode(stateRootMh)
	if err != nil {
		t.Fatalf("header StateRootCID could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedParentMh.Digest, gethHeader.ParentHash.Bytes()) {
		t.Errorf("header state root hash (%x) does not match expected hash (%x)", decodedStateRootMh.Digest, gethHeader.UncleHash.Bytes())
	}

	txRootNode, err := headerNode.LookupByString("TxRootCID")
	if err != nil {
		t.Fatalf("header is missing TxRootCID")
	}
	txRootLink, err := txRootNode.AsLink()
	if err != nil {
		t.Fatalf("header TxRootCID is not a link: %v", err)
	}
	txRootCIDLink, ok := txRootLink.(cidlink.Link)
	if !ok {
		t.Fatalf("header TxRootCID is not a CID: %v", err)
	}
	txRootMh := txRootCIDLink.Hash()
	decodedTxRootMh, err := multihash.Decode(txRootMh)
	if err != nil {
		t.Fatalf("header TxRootCID could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedParentMh.Digest, gethHeader.ParentHash.Bytes()) {
		t.Errorf("header tx root hash (%x) does not match expected hash (%x)", decodedTxRootMh.Digest, gethHeader.UncleHash.Bytes())
	}

	rctRootNode, err := headerNode.LookupByString("RctRootCID")
	if err != nil {
		t.Fatalf("header is missing RctRootCID")
	}
	rctRootLink, err := rctRootNode.AsLink()
	if err != nil {
		t.Fatalf("header RctRootCID is not a link: %v", err)
	}
	rctRootCIDLink, ok := rctRootLink.(cidlink.Link)
	if !ok {
		t.Fatalf("header RctRootCID is not a CID: %v", err)
	}
	rctRootMh := rctRootCIDLink.Hash()
	decodedRctRootMh, err := multihash.Decode(rctRootMh)
	if err != nil {
		t.Fatalf("header RctRootCID could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedParentMh.Digest, gethHeader.ParentHash.Bytes()) {
		t.Errorf("header rct root hash (%x) does not match expected hash (%x)", decodedRctRootMh.Digest, gethHeader.UncleHash.Bytes())
	}

	bloomNode, err := headerNode.LookupByString("Bloom")
	if err != nil {
		t.Fatalf("header is missing Bloom")
	}
	bloomBytes, err := bloomNode.AsBytes()
	if err != nil {
		t.Fatalf("header Bloom should be of type Bytes")
	}
	if !bytes.Equal(bloomBytes, gethHeader.Bloom.Bytes()) {
		t.Errorf("header bloom bytes (%x) does not match expected bytes (%x)", bloomBytes, gethHeader.Bloom.Bytes())
	}

	diffNode, err := headerNode.LookupByString("Difficulty")
	if err != nil {
		t.Fatalf("header is missing Difficulty")
	}
	diffNodeBytes, err := diffNode.AsBytes()
	if err != nil {
		t.Fatalf("header Difficulty should be of type Bytes")
	}
	if !bytes.Equal(diffNodeBytes, gethHeader.Difficulty.Bytes()) {
		t.Errorf("header difficulty (%x) does not match expected difficulty (%x)", diffNodeBytes, gethHeader.Difficulty.Bytes())
	}

	numberNode, err := headerNode.LookupByString("Number")
	if err != nil {
		t.Fatalf("header is missing Number")
	}
	numberBytes, err := numberNode.AsBytes()
	if err != nil {
		t.Fatalf("header Number should be of type Bytes")
	}
	if !bytes.Equal(numberBytes, gethHeader.Number.Bytes()) {
		t.Errorf("header number (%x) does not match expected number (%x)", numberBytes, gethHeader.Number.Bytes())
	}

	gasLimitNode, err := headerNode.LookupByString("GasLimit")
	if err != nil {
		t.Fatalf("header is missing GasLimit")
	}
	gasLimitBytes, err := gasLimitNode.AsBytes()
	if err != nil {
		t.Fatalf("header GasLimit should be of type Bytes")
	}
	gasLimitUint := binary.BigEndian.Uint64(gasLimitBytes)
	if gasLimitUint != gethHeader.GasLimit {
		t.Errorf("header gasLimit (%d) does not match expected gasLimit (%d)", gasLimitUint, gethHeader.GasLimit)
	}

	gasUsedNode, err := headerNode.LookupByString("GasUsed")
	if err != nil {
		t.Fatalf("header is missing GasUsed")
	}
	gasUsedBytes, err := gasUsedNode.AsBytes()
	if err != nil {
		t.Fatalf("header GasUsed should be of type Bytes")
	}
	gasUsedUint := binary.BigEndian.Uint64(gasUsedBytes)
	if gasUsedUint != gethHeader.GasUsed {
		t.Errorf("header gasUsed (%d) does not match expected gasUsed (%d)", gasUsedUint, gethHeader.GasUsed)
	}

	timeNode, err := headerNode.LookupByString("Time")
	if err != nil {
		t.Fatalf("header is missing Time")
	}
	timeBytes, err := timeNode.AsBytes()
	if err != nil {
		t.Fatalf("header GasUsed should be of type Bytes")
	}
	timeUint := binary.BigEndian.Uint64(timeBytes)
	if timeUint != gethHeader.Time {
		t.Errorf("header time (%d) does not match expected time (%d)", timeUint, gethHeader.Time)
	}

	extraNode, err := headerNode.LookupByString("Extra")
	if err != nil {
		t.Fatalf("header is missing Extra")
	}
	extraBytes, err := extraNode.AsBytes()
	if err != nil {
		t.Fatalf("header Extra should be of type Byets")
	}
	if !bytes.Equal(extraBytes, gethHeader.Extra) {
		t.Errorf("header extra bytes (%x) does not match expected bytes (%x)", extraBytes, gethHeader.Extra)
	}

	mixDigestNode, err := headerNode.LookupByString("MixDigest")
	if err != nil {
		t.Fatalf("header is missing MixDigest")
	}
	mixDigestBytes, err := mixDigestNode.AsBytes()
	if err != nil {
		t.Fatalf("header Extra should be of type Byets")
	}
	if !bytes.Equal(mixDigestBytes, gethHeader.MixDigest.Bytes()) {
		t.Errorf("header mixDigest bytes (%x) does not match expected bytes (%x)", mixDigestBytes, gethHeader.MixDigest.Bytes())
	}

	nonceNode, err := headerNode.LookupByString("Nonce")
	if err != nil {
		t.Fatalf("header is missing Nonce")
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		t.Fatalf("header Extra should be of type Byets")
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	if nonce != gethHeader.Nonce.Uint64() {
		t.Errorf("header nonce (%d) does not match expected nonce (%d)", nonce, gethHeader.Nonce.Uint64())
	}
}

func testHeaderEncode(t *testing.T) {
	headerWriter := new(bytes.Buffer)
	if err := header.Encode(headerNode, headerWriter); err != nil {
		t.Fatalf("unable to encode header into writer: %v", err)
	}
	encodedHeaderBytes := headerWriter.Bytes()
	if !bytes.Equal(encodedHeaderBytes, headerRLP) {
		t.Errorf("header encoding (%x) does not match the expected RLP encoding (%x)", encodedHeaderBytes, headerRLP)
	}
	h := new(types.Header)
	if err := header.EncodeHeader(h, headerNode); err != nil {
		t.Fatalf("unable to encode header into geth header: %v", err)
	}
	rlpBy, _ := rlp.EncodeToBytes(h)
	if !bytes.Equal(rlpBy, headerRLP) {
		t.Errorf("header encoding (%x) does not match the expected RLP encoding (%x)", encodedHeaderBytes, headerRLP)
	}
}

func loadBlockFromRLPFile(filename string) (*types.Block, []byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	blockRLP, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	block := new(types.Block)
	return block, blockRLP, rlp.DecodeBytes(blockRLP, block)
}
