package block_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/block"
	"github.com/vulcanize/go-codec-dageth/rct_list"
	"github.com/vulcanize/go-codec-dageth/tx_list"
)

var (
	headerHash      = common.HexToHash("0x4da3b2a073862e40d11acfa94b94f048cdeda19e97e18a99eee43ff9c9c11da0")
	txsHash         = common.HexToHash("0x14b05960bf92af6264a8904706c47cbd9c0c4256f4c9acd00839df607ab5957e")
	rctsHash        = common.HexToHash("0xdf31820e936cdf974bccf5319d99429e6b8780324fd083ad512dffced89a693d")
	blk             = new(block.Block)
	blkCBOREncoding = common.Hex2Bytes("a366486561646572d82a5826000190011b204da3b2a073862e40d11acfa94b94f048cdeda19e97e18a99eee43ff9c9c11da06c5472616e73616374696f6e73d82a582600019c011b2014b05960bf92af6264a8904706c47cbd9c0c4256f4c9acd00839df607ab5957e685265636569707473d82a582600019d011b20df31820e936cdf974bccf5319d99429e6b8780324fd083ad512dffced89a693d")
	blockNode       ipld.Node
)

/*
# Block represents an entire block in the Ethereum blockchain.
type Block struct {
   # CID link to the block at this block
   # This CID is composed of the KECCAK_256 multihash of the RLP encoded block and the EthBlock codec (0x90)
   # Note that the block contains references to the uncles and tx, receipt, and state tries at this height
   HeaderCID       &Header
   # CID link to the list of transactions at this block
   # This CID is composed of the KECCAK_256 multihash of the RLP encoded list of transactions and the EthTxList codec (0x9c)
   TransactionsCID &Transactions
   # CID link to the list of receipts at this block
   # This CID is composed of the KECCAK_256 multihash of the RLP encoded list of receipts and the EthTxReceiptList codec (0x9d)
   ReceiptsCID     &Receipts
}
*/

func TestBlock(t *testing.T) {
	headerMh, err := multihash.Encode(headerHash.Bytes(), multihash.KECCAK_256)
	if err != nil {
		t.Fatalf("unable to derive multihash for header hash: %v", err)
	}
	headerCID := cid.NewCidV1(cid.EthBlock, headerMh)
	blk.Header = append([]byte{0}, headerCID.Bytes()...)

	txsMh, err := multihash.Encode(txsHash.Bytes(), multihash.KECCAK_256)
	if err != nil {
		t.Fatalf("unable to derive multihash for transactions hash: %v", err)
	}
	txsCID := cid.NewCidV1(tx_list.MultiCodecType, txsMh)
	blk.Transactions = append([]byte{0}, txsCID.Bytes()...)

	rctsMh, err := multihash.Encode(rctsHash.Bytes(), multihash.KECCAK_256)
	if err != nil {
		t.Fatalf("unable to derive multihash for receipts hash: %v", err)
	}
	rctsCID := cid.NewCidV1(rct_list.MultiCodecType, rctsMh)
	blk.Receipts = append([]byte{0}, rctsCID.Bytes()...)

	/*
		opts := cbor.CanonicalEncOptions()
		ts := cbor.NewTagSet()
		tsOpts := cbor.TagOptions{
			DecTag: cbor.DecTagRequired,
			EncTag: cbor.EncTagRequired,
		}
		ts.Add(tsOpts, reflect.TypeOf(block.CIDLink{}), 42)
		enc, err := opts.EncModeWithTags(ts)
		if err != nil {
			t.Fatalf("unable to create CBOR marshaller: %v", err)
		}
		blkCBOREncoding, err = enc.Marshal(blk)
		if err != nil {
			t.Fatalf("unable to encode Block to CBOR: %v", err)
		}
	*/

	testBlockDecode(t)
	testBlockNodeContents(t)
	testBlockEncode(t)
}

func testBlockDecode(t *testing.T) {
	blockBuilder := dageth.Type.Block.NewBuilder()
	fmt.Printf("CBOR: %x\r\n", blkCBOREncoding)
	blockReader := bytes.NewReader(blkCBOREncoding)
	if err := dagcbor.Decode(blockBuilder, blockReader); err != nil {
		t.Fatalf("unable to decode block into a DAG-CBOR IPLD node: %v", err)
	}
	blockNode = blockBuilder.Build()
}

func testBlockNodeContents(t *testing.T) {
	headerNode, err := blockNode.LookupByString("Header")
	if err != nil {
		t.Fatalf("block is missing HeaderCID: %v", err)
	}
	headerLink, err := headerNode.AsLink()
	if err != nil {
		t.Fatalf("block HeaderCID is not a link: %v", err)
	}
	headerCIDLink, ok := headerLink.(cidlink.Link)
	if !ok {
		t.Fatalf("block HeaderCID is not a CID: %v", err)
	}
	headerLinkBytes := append([]byte{0}, headerCIDLink.Bytes()...)
	if !bytes.Equal(headerLinkBytes, blk.Header) {
		t.Errorf("block header cid (%x) does not match expected cid (%x)", headerLinkBytes, blk.Header)
	}

	txsNode, err := blockNode.LookupByString("Transactions")
	if err != nil {
		t.Fatalf("block is missing TransactionsCID: %v", err)
	}
	txsLink, err := txsNode.AsLink()
	if err != nil {
		t.Fatalf("block TransactionsCID is not a link: %v", err)
	}
	txsCIDLink, ok := txsLink.(cidlink.Link)
	if !ok {
		t.Fatalf("block TransactionsCID is not a CID: %v", err)
	}
	txsLinkBytes := append([]byte{0}, txsCIDLink.Bytes()...)
	if !bytes.Equal(txsLinkBytes, blk.Transactions) {
		t.Errorf("block txs cid (%x) does not match expected cid (%x)", txsLinkBytes, blk.Transactions)
	}

	rctsNode, err := blockNode.LookupByString("Receipts")
	if err != nil {
		t.Fatalf("block is missing ReceiptsCID: %v", err)
	}
	rctsLink, err := rctsNode.AsLink()
	if err != nil {
		t.Fatalf("block ReceiptsCID is not a link: %v", err)
	}
	rctsCIDLink, ok := rctsLink.(cidlink.Link)
	if !ok {
		t.Fatalf("block ReceiptsCID is not a CID: %v", err)
	}
	rctsLinkBytes := append([]byte{0}, rctsCIDLink.Bytes()...)
	if !bytes.Equal(rctsLinkBytes, blk.Receipts) {
		t.Errorf("block rcts hash (%x) does not match expected hash (%x)", rctsLinkBytes, blk.Receipts)
	}
}

func testBlockEncode(t *testing.T) {
	blockWriter := new(bytes.Buffer)
	if err := dagcbor.Encode(blockNode, blockWriter); err != nil {
		t.Fatalf("unable to encode block into writer: %v", err)
	}
	encodedBlockBytes := blockWriter.Bytes()
	if !bytes.Equal(encodedBlockBytes, blkCBOREncoding) {
		t.Errorf("block encoding (%x) does not match the expected CBOR encoding (%x)", encodedBlockBytes, blkCBOREncoding)
	}
}
