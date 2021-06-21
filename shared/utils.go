package shared

import (
	"fmt"

	"github.com/multiformats/go-multihash"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
)

var evenLeafFlag = []byte{byte(2) << 4}

// RawToCid takes the desired codec and a slice of bytes
// and returns the proper cid of the object.
func RawToCid(codec uint64, rawdata []byte) (cid.Cid, error) {
	c, err := cid.Prefix{
		Codec:    codec,
		Version:  1,
		MhType:   multihash.KECCAK_256,
		MhLength: -1,
	}.Sum(rawdata)
	if err != nil {
		return cid.Cid{}, err
	}
	return c, nil
}

// Keccak256ToCid takes a keccak256 hash and returns its cid based on the codec given.
func Keccak256ToCid(codec uint64, h []byte) cid.Cid {
	buf, err := multihash.Encode(h, multihash.KECCAK_256)
	if err != nil {
		panic(err)
	}

	return cid.NewCidV1(codec, multihash.Multihash(buf))
}

// AddressToLeafKey hashes an returns an address
func AddressToLeafKey(address common.Address) []byte {
	return crypto.Keccak256(address[:])
}

// AddressToEncodedPath hashes an address and appends the even-number leaf flag to it
func AddressToEncodedPath(address common.Address) []byte {
	addrHash := crypto.Keccak256(address[:])
	decodedPath := append(evenLeafFlag, addrHash...)
	return decodedPath
}

// GetTxType returns the eth tx type
func GetTxType(node ipld.Node) (uint8, error) {
	tyNode, err := node.LookupByString("Type")
	if err != nil {
		return 0, err
	}
	tyBytes, err := tyNode.AsBytes()
	if err != nil {
		return 0, err
	}
	if len(tyBytes) != 1 {
		return 0, fmt.Errorf("tx type should be a single byte")
	}
	return tyBytes[0], nil
}

type WriteableByteSlice struct {
	enc *[]byte
}

func NewWriteableByteSlice(enc *[]byte) WriteableByteSlice {
	return WriteableByteSlice{enc: enc}
}

func (w WriteableByteSlice) Write(b []byte) (int, error) {
	*w.enc = append(*w.enc, b...)
	return len(b), nil
}
