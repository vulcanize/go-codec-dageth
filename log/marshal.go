package log

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
)

// Encode provides an IPLD codec encode interface for eth log IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code TBD when this package is invoked via init.
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
	log := new(types.Log)
	if err := EncodeLog(log, inNode); err != nil {
		return nil, err
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	if err := rlp.Encode(wbs, log); err != nil {
		return nil, err
	}
	return enc, nil
}

// EncodeLog packs the node into the go-ethereum Log
func EncodeLog(log *types.Log, inNode ipld.Node) error {
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Log.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return err
	}
	node := builder.Build()
	for _, pFunc := range requiredPackFuncs {
		if err := pFunc(log, node); err != nil {
			return fmt.Errorf("invalid DAG-ETH Log form (%v)", err)
		}
	}
	return nil
}

var requiredPackFuncs = []func(*types.Log, ipld.Node) error{
	packAddress,
	packTopics,
	packData,
}

func packAddress(log *types.Log, node ipld.Node) error {
	addrNode, err := node.LookupByString("Address")
	if err != nil {
		return fmt.Errorf("receipt log is missing an Address node: %v", err)
	}
	addrBytes, err := addrNode.AsBytes()
	if err != nil {
		return err
	}
	log.Address = common.BytesToAddress(addrBytes)
	return nil
}

func packTopics(log *types.Log, node ipld.Node) error {
	topicsNode, err := node.LookupByString("Topics")
	if err != nil {
		return fmt.Errorf("receipt log is missing a Topics node: %v", err)
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
	log.Topics = topics
	return nil
}

func packData(log *types.Log, node ipld.Node) error {
	dataNode, err := node.LookupByString("Data")
	if err != nil {
		return fmt.Errorf("receipt log is missing a Data node: %v", err)
	}
	data, err := dataNode.AsBytes()
	if err != nil {
		return err
	}
	log.Data = data
	return nil
}
