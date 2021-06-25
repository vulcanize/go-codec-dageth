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
	tyNode, err := node.LookupByString("TxType")
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

// HexToCompact converts a hex path to the compact encoded format
func HexToCompact(hex []byte) []byte {
	return hexToCompact(hex)
}

func hexToCompact(hex []byte) []byte {
	terminator := byte(0)
	if hasTerm(hex) {
		terminator = 1
		hex = hex[:len(hex)-1]
	}
	buf := make([]byte, len(hex)/2+1)
	buf[0] = terminator << 5 // the flag byte
	if len(hex)&1 == 1 {
		buf[0] |= 1 << 4 // odd flag
		buf[0] |= hex[0] // first nibble is contained in the first byte
		hex = hex[1:]
	}
	decodeNibbles(hex, buf[1:])
	return buf
}

func decodeNibbles(nibbles []byte, bytes []byte) {
	for bi, ni := 0, 0; ni < len(nibbles); bi, ni = bi+1, ni+2 {
		bytes[bi] = nibbles[ni]<<4 | nibbles[ni+1]
	}
}

// CompactToHex converts a compact encoded path to hex format
func CompactToHex(compact []byte) []byte {
	return compactToHex(compact)
}

func compactToHex(compact []byte) []byte {
	if len(compact) == 0 {
		return compact
	}
	base := keybytesToHex(compact)
	// delete terminator flag
	if base[0] < 2 {
		base = base[:len(base)-1]
	}
	// apply odd flag
	chop := 2 - base[0]&1
	return base[chop:]
}

func keybytesToHex(str []byte) []byte {
	l := len(str)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	nibbles[l-1] = 16
	return nibbles
}

// hasTerm returns whether a hex key has the terminator flag.
func hasTerm(s []byte) bool {
	return len(s) > 0 && s[len(s)-1] == 16
}
