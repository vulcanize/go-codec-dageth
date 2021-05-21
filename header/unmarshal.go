package dageth_header

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"
	"google.golang.org/protobuf/encoding/protowire"
)

// ErrIntOverflow is returned a varint overflows during decode, it indicates
// malformed data
var ErrIntOverflow = fmt.Errorf("protobuf: varint overflow")

var (
	MultiCodecType = uint64(cid.EthBlock)
	MultiHashType = uint64(multihash.KECCAK_224)
)

// Decode provides an IPLD codec decode interface for ETH header IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x90 when this package is invoked via init.
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

/*
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
	    Time Uint
	    Extra Bytes
	    MixDigest Hash
	    Nonce BlockNonce
	}
*/

// DecodeBytes is like Decode, but it uses an input buffer directly.
// Decode will grab or read all the bytes from an io.Reader anyway, so this can
// save having to copy the bytes or create a bytes.Buffer.
func DecodeBytes(na ipld.NodeAssembler, src []byte) error {
	var header types.Header
	if err := rlp.DecodeBytes(src, &header); err != nil {
		return err
	}
	ma, err := na.BeginMap(15)
	if err != nil {
		return err
	}
	for _, upFunc := range requiredUnpackFuncs {
		if err := upFunc(ma, header); err != nil {
			return fmt.Errorf("invalid DAG-ETH Header binary (%v)", err)
		}
	}
	return ma.Finish()
}

var requiredUnpackFuncs = []func(ma ipld.MapAssembler, header types.Header) error {
	unpackParentCID,
	unpackUnclesCID,
	unpackCoinbase,
	unpackStateRootCID,
	unpackTxRootCID,
	unpackRctRootCID,
	unpackBloom,
	unpackDifficulty,
	unpackNumber,
	unpackGasLimit,
	unpackGasUsed,
	unpackTime,
	unpackExtra,
	unpackMixDigest,
	unpackNonce,
}

func unpackNonce(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("Nonce"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.Nonce[:]); err != nil {
		return err
	}
	return nil
}

func unpackMixDigest(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("MixDigest"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.MixDigest.Bytes()); err != nil {
		return err
	}
	return nil
}

func unpackExtra(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("Extra"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.Extra); err != nil {
		return err
	}
	return nil
}

func unpackTime(ma ipld.MapAssembler, header types.Header) error {
	timeBytes := make([]byte, 0, 8)
	binary.BigEndian.PutUint64(timeBytes, header.Time)
	if err := ma.AssembleKey().AssignString("Time"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(timeBytes); err != nil {
		return err
	}
	return nil
}

func unpackGasUsed(ma ipld.MapAssembler, header types.Header) error {
	gasUsedBytes := make([]byte, 0, 8)
	binary.BigEndian.PutUint64(gasUsedBytes, header.GasUsed)
	if err := ma.AssembleKey().AssignString("GasUsed"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(gasUsedBytes); err != nil {
		return err
	}
	return nil
}

func unpackGasLimit(ma ipld.MapAssembler, header types.Header) error {
	gasLimitBytes := make([]byte, 0, 8)
	binary.BigEndian.PutUint64(gasLimitBytes, header.GasLimit)
	if err := ma.AssembleKey().AssignString("GasLimit"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(gasLimitBytes); err != nil {
		return err
	}
	return nil
}

func unpackNumber(ma ipld.MapAssembler, header types.Header) error {
	if header.Number == nil {
		return fmt.Errorf("header cannot have `nil` Number")
	}
	if err := ma.AssembleKey().AssignString("Number"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.Number.Bytes()); err != nil {
		return err
	}
	return nil
}

func unpackDifficulty(ma ipld.MapAssembler, header types.Header) error {
	if header.Difficulty == nil {
		return fmt.Errorf("header cannot have `nil` Difficulty")
	}
	if err := ma.AssembleKey().AssignString("Difficulty"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.Difficulty.Bytes()); err != nil {
		return err
	}
	return nil
}

func unpackBloom(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("Bloom"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.Bloom.Bytes()); err != nil {
		return err
	}
	return nil
}

func unpackRctRootCID(ma ipld.MapAssembler, header types.Header) error {
	rctMh, err := multihash.Encode(header.ReceiptHash.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	rctCID := cid.NewCidV1(MultiCodecType, rctMh)
	rctLinkCID := cidlink.Link{Cid: rctCID}
	if err := ma.AssembleKey().AssignString("RctRootCID"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignLink(rctLinkCID); err != nil {
		return err
	}
	return nil
}

func unpackTxRootCID(ma ipld.MapAssembler, header types.Header) error {
	txMh, err := multihash.Encode(header.TxHash.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	txCID := cid.NewCidV1(MultiCodecType, txMh)
	txLinkCID := cidlink.Link{Cid: txCID}
	if err := ma.AssembleKey().AssignString("TxRootCID"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignLink(txLinkCID); err != nil {
		return err
	}
	return nil
}

func unpackStateRootCID(ma ipld.MapAssembler, header types.Header) error {
	srMh, err := multihash.Encode(header.Root.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	srCID := cid.NewCidV1(MultiCodecType, srMh)
	srLinkCID := cidlink.Link{Cid: srCID}
	if err := ma.AssembleKey().AssignString("StateRootCID"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignLink(srLinkCID); err != nil {
		return err
	}
	return nil
}

func unpackCoinbase(ma ipld.MapAssembler, header types.Header) error {
	if err := ma.AssembleKey().AssignString("Coinbase"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(header.Coinbase.Bytes()); err != nil {
		return err
	}
	return nil
}

func unpackUnclesCID(ma ipld.MapAssembler, header types.Header) error {
	unclesMh, err := multihash.Encode(header.UncleHash.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	unclesCID := cid.NewCidV1(MultiCodecType, unclesMh)
	unclesLinkCID := cidlink.Link{Cid: unclesCID}
	if err := ma.AssembleKey().AssignString("UnclesCID"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignLink(unclesLinkCID); err != nil {
		return err
	}
	return nil
}

func unpackParentCID(ma ipld.MapAssembler, header types.Header) error {
	parentMh, err := multihash.Encode(header.ParentHash.Bytes(), MultiHashType)
	if err != nil {
		return err
	}
	parentCID := cid.NewCidV1(MultiCodecType, parentMh)
	parentLinkCID := cidlink.Link{Cid: parentCID}
	if err := ma.AssembleKey().AssignString("ParentCID"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignLink(parentLinkCID); err != nil {
		return err
	}
	return nil
}

// DecodeBytes is like Decode, but it uses an input buffer directly.
// Decode will grab or read all the bytes from an io.Reader anyway, so this can
// save having to copy the bytes or create a bytes.Buffer.
func DecodeBytes(na ipld.NodeAssembler, src []byte) error {
	remaining := src

	ma, err := na.BeginMap(2)
	if err != nil {
		return err
	}
	var links ipld.ListAssembler

	haveData := false
	haveLinks := false
	for {
		if len(remaining) == 0 {
			break
		}

		fieldNum, wireType, n := protowire.ConsumeTag(remaining)
		if n < 0 {
			return protowire.ParseError(n)
		}
		remaining = remaining[n:]

		if wireType != 2 {
			return fmt.Errorf("protobuf: (PBNode) invalid wireType, expected 2, got %d", wireType)
		}

		// Note that we allow Data and Links to come in either order,
		// since the spec defines that decoding "should" accept either form.
		// This is for backwards compatibility with older IPFS data.

		switch fieldNum {
		case 1:
			if haveData {
				return fmt.Errorf("protobuf: (PBNode) duplicate Data section")
			}

			chunk, n := protowire.ConsumeBytes(remaining)
			if n < 0 {
				return protowire.ParseError(n)
			}
			remaining = remaining[n:]

			if links != nil {
				// Links came before Data.
				// Finish them before we start Data.
				if err := links.Finish(); err != nil {
					return err
				}
				links = nil
			}

			if err := ma.AssembleKey().AssignString("Data"); err != nil {
				return err
			}
			if err := ma.AssembleValue().AssignBytes(chunk); err != nil {
				return err
			}
			haveData = true

		case 2:
			bytesLen, n := protowire.ConsumeVarint(remaining)
			if n < 0 {
				return protowire.ParseError(n)
			}
			remaining = remaining[n:]

			if links == nil {
				if haveLinks {
					return fmt.Errorf("protobuf: (PBNode) duplicate Links section")
				}

				// The repeated "Links" part begins.
				if err := ma.AssembleKey().AssignString("Links"); err != nil {
					return err
				}
				links, err = ma.AssembleValue().BeginList(0)
				if err != nil {
					return err
				}
			}

			curLink, err := links.AssembleValue().BeginMap(3)
			if err != nil {
				return err
			}
			if err := unmarshalLink(remaining[:bytesLen], curLink); err != nil {
				return err
			}
			remaining = remaining[bytesLen:]
			if err := curLink.Finish(); err != nil {
				return err
			}
			haveLinks = true

		default:
			return fmt.Errorf("protobuf: (PBNode) invalid fieldNumber, expected 1 or 2, got %d", fieldNum)
		}
	}

	if links != nil {
		// We had some links at the end, so finish them.
		if err := links.Finish(); err != nil {
			return err
		}

	} else if !haveLinks {
		// We didn't have any links.
		// Since we always want a Links field, add one here.
		if err := ma.AssembleKey().AssignString("Links"); err != nil {
			return err
		}
		links, err := ma.AssembleValue().BeginList(0)
		if err != nil {
			return err
		}
		if err := links.Finish(); err != nil {
			return err
		}
	}
	return ma.Finish()
}

func unmarshalLink(remaining []byte, ma ipld.MapAssembler) error {
	haveHash := false
	haveName := false
	haveTsize := false
	for {
		if len(remaining) == 0 {
			break
		}

		fieldNum, wireType, n := protowire.ConsumeTag(remaining)
		if n < 0 {
			return protowire.ParseError(n)
		}
		remaining = remaining[n:]

		switch fieldNum {
		case 1:
			if haveHash {
				return fmt.Errorf("protobuf: (PBLink) duplicate Hash section")
			}
			if haveName {
				return fmt.Errorf("protobuf: (PBLink) invalid order, found Name before Hash")
			}
			if haveTsize {
				return fmt.Errorf("protobuf: (PBLink) invalid order, found Tsize before Hash")
			}
			if wireType != 2 {
				return fmt.Errorf("protobuf: (PBLink) wrong wireType (%d) for Hash", wireType)
			}

			chunk, n := protowire.ConsumeBytes(remaining)
			if n < 0 {
				return protowire.ParseError(n)
			}
			remaining = remaining[n:]

			_, c, err := cid.CidFromBytes(chunk)
			if err != nil {
				return fmt.Errorf("invalid Hash field found in link, expected CID (%v)", err)
			}
			if err := ma.AssembleKey().AssignString("Hash"); err != nil {
				return err
			}
			if err := ma.AssembleValue().AssignLink(cidlink.Link{Cid: c}); err != nil {
				return err
			}
			haveHash = true

		case 2:
			if haveName {
				return fmt.Errorf("protobuf: (PBLink) duplicate Name section")
			}
			if haveTsize {
				return fmt.Errorf("protobuf: (PBLink) invalid order, found Tsize before Name")
			}
			if wireType != 2 {
				return fmt.Errorf("protobuf: (PBLink) wrong wireType (%d) for Name", wireType)
			}

			chunk, n := protowire.ConsumeBytes(remaining)
			if n < 0 {
				return protowire.ParseError(n)
			}
			remaining = remaining[n:]

			if err := ma.AssembleKey().AssignString("Name"); err != nil {
				return err
			}
			if err := ma.AssembleValue().AssignString(string(chunk)); err != nil {
				return err
			}
			haveName = true

		case 3:
			if haveTsize {
				return fmt.Errorf("protobuf: (PBLink) duplicate Tsize section")
			}
			if wireType != 0 {
				return fmt.Errorf("protobuf: (PBLink) wrong wireType (%d) for Tsize", wireType)
			}

			v, n := protowire.ConsumeVarint(remaining)
			if n < 0 {
				return protowire.ParseError(n)
			}
			remaining = remaining[n:]

			if err := ma.AssembleKey().AssignString("Tsize"); err != nil {
				return err
			}
			if err := ma.AssembleValue().AssignInt(int64(v)); err != nil {
				return err
			}
			haveTsize = true

		default:
			return fmt.Errorf("protobuf: (PBLink) invalid fieldNumber, expected 1, 2 or 3, got %d", fieldNum)
		}
	}

	if !haveHash {
		return fmt.Errorf("invalid Hash field found in link, expected CID")
	}

	return nil
}
