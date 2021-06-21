package rct

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
)

// Encode provides an IPLD codec encode interface for eth receipt IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x95 when this package is invoked via init.
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
	rct := new(receiptRLP)
	txType, err := packReceiptRLP(rct, inNode)
	if err != nil {
		return enc, fmt.Errorf("unable to encode receiptRLP (%v)", err)
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	switch txType {
	case types.LegacyTxType:
		if err := rlp.Encode(wbs, rct); err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Receipt form (%v)", err)
		}
		return enc, nil
	case types.AccessListTxType:
		enc = append(enc, txType)
		if err := rlp.Encode(wbs, rct); err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Receipt form (%v)", err)
		}
		return enc, nil
	default:
		return enc, fmt.Errorf("invalid DAG-ETH Receipt form (unrecognized TxType %d)", txType)
	}
}

var (
	receiptStatusFailedRLP     = []byte{}
	receiptStatusSuccessfulRLP = []byte{0x01}
)

// EncodeReceipt packs the node into the go-ethereum Receipt
func EncodeReceipt(receipt *types.Receipt, inNode ipld.Node) error {
	rct := new(receiptRLP)
	txType, err := packReceiptRLP(rct, inNode)
	if err != nil {
		return err
	}
	receipt.Type = txType
	receipt.Bloom = rct.Bloom
	receipt.CumulativeGasUsed = rct.CumulativeGasUsed
	receipt.Logs = rct.Logs
	switch {
	case bytes.Equal(rct.PostStateOrStatus, receiptStatusSuccessfulRLP):
		receipt.Status = types.ReceiptStatusSuccessful
	case bytes.Equal(rct.PostStateOrStatus, receiptStatusFailedRLP):
		receipt.Status = types.ReceiptStatusFailed
	case len(rct.PostStateOrStatus) == len(common.Hash{}):
		receipt.PostState = rct.PostStateOrStatus
	default:
		return fmt.Errorf("invalid DAG-ETH Receipt PostStateOrStatus %x", rct.PostStateOrStatus)
	}
	return nil
}

func packReceiptRLP(rct *receiptRLP, inNode ipld.Node) (uint8, error) {
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Receipt.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return 0, err
	}
	node := builder.Build()
	txType, err := shared.GetTxType(node)
	if err != nil {
		return 0, fmt.Errorf("invalid DAG-ETH Receipt form (%v)", err)
	}
	for _, pFunc := range requiredPackFuncs {
		if err := pFunc(rct, node); err != nil {
			return 0, fmt.Errorf("invalid DAG-ETH Receipt form (%v)", err)
		}
	}
	return txType, nil
}

// the consensus struct for a receipt is not an exported type from go-ethereum
// so until types.Receipt has a MarshalBinary method we will pack and RLP encode a custom struct
type receiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             types.Bloom
	Logs              []*types.Log
}

var requiredPackFuncs = []func(*receiptRLP, ipld.Node) error{
	packPostStateOrStatus,
	packCumulativeGasUsed,
	packBloom,
	packLogs,
}

func packPostStateOrStatus(rct *receiptRLP, node ipld.Node) error {
	psNode, err := node.LookupByString("PostState")
	if err != nil {
		return err
	}
	if !psNode.IsNull() {
		psBytes, err := psNode.AsBytes()
		if err != nil {
			return err
		}
		rct.PostStateOrStatus = psBytes
		return nil
	}

	sNode, err := node.LookupByString("Status")
	if err != nil {
		return err
	}
	if sNode.IsNull() {
		return fmt.Errorf("receipt Node must have either PostState or Status")
	}
	sBytes, err := sNode.AsBytes()
	if err != nil {
		return err
	}
	rct.PostStateOrStatus = sBytes
	return nil
}

func packCumulativeGasUsed(rct *receiptRLP, node ipld.Node) error {
	cguNode, err := node.LookupByString("CumulativeGasUsed")
	if err != nil {
		return err
	}
	cguBytes, err := cguNode.AsBytes()
	if err != nil {
		return err
	}
	rct.CumulativeGasUsed = binary.BigEndian.Uint64(cguBytes)
	return nil
}

func packBloom(rct *receiptRLP, node ipld.Node) error {
	bloomNode, err := node.LookupByString("Bloom")
	if err != nil {
		return err
	}
	bloomBytes, err := bloomNode.AsBytes()
	if err != nil {
		return err
	}
	rct.Bloom = types.BytesToBloom(bloomBytes)
	return nil
}

func packLogs(rct *receiptRLP, node ipld.Node) error {
	logsNode, err := node.LookupByString("Logs")
	if err != nil {
		return err
	}
	logs := make([]*types.Log, logsNode.Length())
	logsIt := logsNode.ListIterator()
	for !logsIt.Done() {
		logIndex, logNode, err := logsIt.Next()
		if err != nil {
			return err
		}
		addrNode, err := logNode.LookupByString("Address")
		if err != nil {
			return err
		}
		addrBytes, err := addrNode.AsBytes()
		if err != nil {
			return err
		}
		topicsNode, err := logNode.LookupByString("Topics")
		if err != nil {
			return err
		}
		topics := make([]common.Hash, topicsNode.Length())
		topicsIt := topicsNode.ListIterator()
		for !topicsIt.Done() {
			topicIndex, topicNode, err := topicsIt.Next()
			if err != nil {
				return err
			}
			topicBytes, err := topicNode.AsBytes()
			if err != nil {
				return err
			}
			topics[topicIndex] = common.BytesToHash(topicBytes)
		}
		dataNode, err := logNode.LookupByString("Data")
		if err != nil {
			return err
		}
		data, err := dataNode.AsBytes()
		if err != nil {
			return err
		}
		logs[logIndex] = &types.Log{
			Address: common.BytesToAddress(addrBytes),
			Topics:  topics,
			Data:    data,
		}
	}
	rct.Logs = logs
	return nil
}
