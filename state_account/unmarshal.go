package dageth_account

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"
)

// Decode provides an IPLD codec decode interface for eth state account IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x97 when this package is invoked via init.
func Decode(na ipld.NodeAssembler, in io.Reader) error {
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
	return DecodeBytes(na, src)
}

// DecodeBytes is like Decode, but it uses an input buffer directly.
// Decode will grab or read all the bytes from an io.Reader anyway, so this can
// save having to copy the bytes or create a bytes.Buffer.
func DecodeBytes(na ipld.NodeAssembler, src []byte) error {
	var account state.Account
	if err := rlp.DecodeBytes(src, &account); err != nil {
		return err
	}
	return DecodeAccount(na, account)
}

// DecodeAccount unpacks a go-ethereum Account into a NodeAssembler
func DecodeAccount(na ipld.NodeAssembler, header state.Account) error {
	ma, err := na.BeginMap(15)
	if err != nil {
		return err
	}
	for _, upFunc := range requiredUnpackFuncs {
		if err := upFunc(ma, header); err != nil {
			return fmt.Errorf("invalid DAG-ETH Account binary (%v)", err)
		}
	}
	return ma.Finish()
}

var requiredUnpackFuncs = []func(ipld.MapAssembler, state.Account) error{
	unpackNonce,
	unpackBalance,
	unpackStorageRootCID,
	unpackCodeCID,
}

func unpackNonce(ma ipld.MapAssembler, account state.Account) error {
	if err := ma.AssembleKey().AssignString("Nonce"); err != nil {
		return err
	}
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, account.Nonce)
	if err := ma.AssembleValue().AssignBytes(nonceBytes); err != nil {
		return err
	}
	return nil
}

func unpackBalance(ma ipld.MapAssembler, account state.Account) error {
	if account.Balance == nil {
		return fmt.Errorf("account balance cannot be null")
	}
	if err := ma.AssembleKey().AssignString("Balance"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(account.Balance.Bytes()); err != nil {
		return err
	}
	return nil
}

func unpackStorageRootCID(ma ipld.MapAssembler, account state.Account) error {
	srMh, err := multihash.Encode(account.Root.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	srCID := cid.NewCidV1(cid.EthStorageTrie, srMh)
	srLinkCID := cidlink.Link{Cid: srCID}
	if err := ma.AssembleKey().AssignString("StorageRootCID"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignLink(srLinkCID)
}

func unpackCodeCID(ma ipld.MapAssembler, account state.Account) error {
	cMh, err := multihash.Encode(account.Root.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	cCID := cid.NewCidV1(cid.Raw, cMh)
	cLinkCID := cidlink.Link{Cid: cCID}
	if err := ma.AssembleKey().AssignString("CodeCID"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignLink(cLinkCID)
}
