package account_test

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	account "github.com/vulcanize/go-codec-dageth/state_account"
)

var (
	emptyStateRootNodeRLP, _ = rlp.EncodeToBytes([]byte{})
	mockStateRoot            = crypto.Keccak256Hash(emptyStateRootNodeRLP)
	emptyCodeHash            = crypto.Keccak256Hash([]byte{}).Bytes()
	mockAccount              = &state.Account{
		Root:     mockStateRoot,
		Balance:  big.NewInt(1000000000),
		CodeHash: emptyCodeHash,
		Nonce:    1,
	}
	accountRLP  []byte
	accountNode ipld.Node
)

/* IPLD Schemas
type ByteCode bytes

type Account struct {
	Nonce    Uint
	Balance  Balance
	StorageRootCID &StorageTrieNode
	CodeCID &ByteCode
}
*/

func TestAccountCodec(t *testing.T) {
	var err error
	accountRLP, err = rlp.EncodeToBytes(mockAccount)
	if err != nil {
		t.Fatalf("unable to RLP encode state account: %v", err)
	}
	testAccountDecode(t)
	testAccountNodeContents(t)
	testAccountEncode(t)
}

func testAccountDecode(t *testing.T) {
	accountBuilder := dageth.Type.Account.NewBuilder()
	accountReader := bytes.NewReader(accountRLP)
	if err := account.Decode(accountBuilder, accountReader); err != nil {
		t.Fatalf("unable to decode account into an IPLD node: %v", err)
	}
	accountNode = accountBuilder.Build()
}

func testAccountNodeContents(t *testing.T) {
	stateRootNode, err := accountNode.LookupByString("StorageRootCID")
	if err != nil {
		t.Fatalf("account is missing StorageRootCID: %v", err)
	}
	srLink, err := stateRootNode.AsLink()
	if err != nil {
		t.Fatalf("account StorageRootCID is not a link: %v", err)
	}
	srCIDLink, ok := srLink.(cidlink.Link)
	if !ok {
		t.Fatalf("account StorageRootCID is not a CID: %v", err)
	}
	srMh := srCIDLink.Hash()
	decodedSrMh, err := multihash.Decode(srMh)
	if err != nil {
		t.Fatalf("account StorageRootCID could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedSrMh.Digest, mockAccount.Root.Bytes()) {
		t.Errorf("account state root hash (%x) does not match expected hash (%x)", decodedSrMh.Digest, mockAccount.Root.Bytes())
	}

	balanceNode, err := accountNode.LookupByString("Balance")
	if err != nil {
		t.Fatalf("account is missing Balance %v", err)
	}
	balanceBytes, err := balanceNode.AsBytes()
	if err != nil {
		t.Fatalf("account Balance should be of type Bytes %v", err)
	}
	if !bytes.Equal(balanceBytes, mockAccount.Balance.Bytes()) {
		t.Errorf("account balance (%x) does not match expected balance (%x)", balanceBytes, mockAccount.Balance.Bytes())
	}

	nonceNode, err := accountNode.LookupByString("Nonce")
	if err != nil {
		t.Fatalf("account is missing Balance %v", err)
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		t.Fatalf("account Balance should be of type Bytes %v", err)
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	if nonce != mockAccount.Nonce {
		t.Errorf("account nonce (%d) does not match expected nonce (%d)", nonce, mockAccount.Nonce)
	}

	codeNode, err := accountNode.LookupByString("CodeCID")
	if err != nil {
		t.Fatalf("account is missing CodeCID: %v", err)
	}
	codeLink, err := codeNode.AsLink()
	if err != nil {
		t.Fatalf("account CodeCID is not a link: %v", err)
	}
	codeCIDLink, ok := codeLink.(cidlink.Link)
	if !ok {
		t.Fatalf("account CodeCID is not a CID: %v", err)
	}
	codeMultihash := codeCIDLink.Hash()
	decodedCodeMulithash, err := multihash.Decode(codeMultihash)
	if err != nil {
		t.Fatalf("account CodeCID could not be decoded into multihash: %v", err)
	}
	if !bytes.Equal(decodedCodeMulithash.Digest, mockAccount.CodeHash) {
		t.Errorf("account code hash (%x) does not match expected hash (%x)", decodedCodeMulithash.Digest, mockAccount.CodeHash)
	}
}

func testAccountEncode(t *testing.T) {
	accountWriter := new(bytes.Buffer)
	if err := account.Encode(accountNode, accountWriter); err != nil {
		t.Fatalf("unable to encode state account into writer: %v", err)
	}
	encodedAccountBytes := accountWriter.Bytes()
	if !bytes.Equal(encodedAccountBytes, accountRLP) {
		t.Errorf("state account encoding (%x) does not match the expected RLP encoding (%x)", encodedAccountBytes, accountRLP)
	}
}
