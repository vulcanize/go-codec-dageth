package rct

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ipld/go-ipld-prime"
)

// Decode provides an IPLD codec decode interface for eth log IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code TBD when this package is invoked via init.
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
	var log types.Log
	if err := rlp.DecodeBytes(src, log); err != nil {
		return err
	}
	return DecodeLog(na, log)
}

// DecodeLog unpacks a go-ethereum Log into the NodeAssembler
func DecodeLog(na ipld.NodeAssembler, log types.Log) error {
	ma, err := na.BeginMap(3)
	if err != nil {
		return err
	}
	for _, upFunc := range requiredUnpackFuncs {
		if err := upFunc(ma, log); err != nil {
			return fmt.Errorf("invalid DAG-ETH Log binary (%v)", err)
		}
	}
	return ma.Finish()
}

var requiredUnpackFuncs = []func(ipld.MapAssembler, types.Log) error{
	unpackAddress,
	unpackTopics,
	unpackData,
}

func unpackAddress(ma ipld.MapAssembler, log types.Log) error {
	if err := ma.AssembleKey().AssignString("Address"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(log.Address.Bytes())
}

func unpackTopics(ma ipld.MapAssembler, log types.Log) error {
	if err := ma.AssembleKey().AssignString("Topics"); err != nil {
		return err
	}
	topicsLa, err := ma.AssembleValue().BeginList(int64(len(log.Topics)))
	if err != nil {
		return err
	}
	for _, topic := range log.Topics {
		if err := topicsLa.AssembleValue().AssignBytes(topic.Bytes()); err != nil {
			return err
		}
	}
	return topicsLa.Finish()
}

func unpackData(ma ipld.MapAssembler, log types.Log) error {
	if err := ma.AssembleKey().AssignString("Data"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(log.Data)
}
