package account

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
)

// Encode provides an IPLD codec encode interface for eth state account IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x97 when this package is invoked via init.
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
	account := new(state.Account)
	if err := EncodeAccount(account, inNode); err != nil {
		return enc, err
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	if err := rlp.Encode(wbs, account); err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Account form (unable to RLP encode account: %v)", err)
	}
	return enc, nil
}

// EncodeAccount packs the node into the provided go-ethereum Account
func EncodeAccount(header *state.Account, inNode ipld.Node) error {
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Account.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return err
	}
	node := builder.Build()
	for _, pFunc := range requiredPackFuncs {
		if err := pFunc(header, node); err != nil {
			return fmt.Errorf("invalid DAG-ETH Account form (%v)", err)
		}
	}
	return nil
}

var requiredPackFuncs = []func(*state.Account, ipld.Node) error{
	packNonce,
	packBalance,
	packStorageRootCID,
	packCodeCID,
}

func packNonce(account *state.Account, node ipld.Node) error {
	n, err := node.LookupByString("Nonce")
	if err != nil {
		return err
	}
	nBytes, err := n.AsBytes()
	if err != nil {
		return err
	}
	account.Nonce = binary.BigEndian.Uint64(nBytes)
	return nil
}

func packBalance(account *state.Account, node ipld.Node) error {
	b, err := node.LookupByString("Balance")
	if err != nil {
		return err
	}
	bBytes, err := b.AsBytes()
	if err != nil {
		return err
	}
	account.Balance = new(big.Int).SetBytes(bBytes)
	return nil
}

func packStorageRootCID(account *state.Account, node ipld.Node) error {
	srCID, err := node.LookupByString("StorageRootCID")
	if err != nil {
		return err
	}
	srLink, err := srCID.AsLink()
	if err != nil {
		return err
	}
	srCIDLink, ok := srLink.(cidlink.Link)
	if !ok {
		return fmt.Errorf("account must have a StateRootCID")
	}
	srMh := srCIDLink.Hash()
	decodedSrMh, err := multihash.Decode(srMh)
	if err != nil {
		return fmt.Errorf("unable to decode StorageRootCID multihash: %v", err)
	}
	account.Root = common.BytesToHash(decodedSrMh.Digest)
	return nil
}

func packCodeCID(account *state.Account, node ipld.Node) error {
	cCID, err := node.LookupByString("CodeCID")
	if err != nil {
		return err
	}
	cLink, err := cCID.AsLink()
	if err != nil {
		return err
	}
	cCIDLink, ok := cLink.(cidlink.Link)
	if !ok {
		return fmt.Errorf("account must have a CodeCID")
	}
	cMh := cCIDLink.Hash()
	decodedCMh, err := multihash.Decode(cMh)
	if err != nil {
		return fmt.Errorf("unable to decode CodeCID multihash: %v", err)
	}
	account.CodeHash = decodedCMh.Digest
	return nil
}
