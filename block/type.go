package block

type CIDLink []byte

type Block struct {
	// HeaderCID links to the header for this block
	Header CIDLink `cbor:"Header"`
	// TransactionsCID links to the list of transactions for this block (not the trie, but the complete RLP encoded list)
	Transactions CIDLink `cbor:"Transactions"`
	// ReceiptsCID links to the list of receipts for this block (not the trie, but the comple RLP encoded list)
	Receipts CIDLink `cbor:"Receipts"`
}
