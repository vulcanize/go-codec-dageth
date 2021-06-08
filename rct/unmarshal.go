package rct

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ipld/go-ipld-prime"
)

// Decode provides an IPLD codec decode interface for eth receipt IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x95 when this package is invoked via init.
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
	var rct types.Receipt
	if err := rct.UnmarshalBinary(src); err != nil {
		return err
	}
	return DecodeReceipt(na, rct)
}

// DecodeReceipt unpacks a go-ethereum Receipt into the NodeAssembler
func DecodeReceipt(na ipld.NodeAssembler, receipt types.Receipt) error {
	ma, err := na.BeginMap(5)
	if err != nil {
		return err
	}
	for _, upFunc := range requiredUnpackFuncs {
		if err := upFunc(ma, receipt); err != nil {
			return fmt.Errorf("invalid DAG-ETH Receipt binary (%v)", err)
		}
	}
	return ma.Finish()
}

var requiredUnpackFuncs = []func(ipld.MapAssembler, types.Receipt) error{
	unpackTxType,
	unpackPostStateOrStatus,
	unpackCumulativeGasUsed,
	unpackBloom,
	unpackLogs,
}

func unpackTxType(ma ipld.MapAssembler, rct types.Receipt) error {
	if err := ma.AssembleKey().AssignString("Type"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes([]byte{rct.Type})
}

func unpackPostStateOrStatus(ma ipld.MapAssembler, rct types.Receipt) error {
	if len(rct.PostState) > 0 {
		if err := ma.AssembleKey().AssignString("PostState"); err != nil {
			return err
		}
		if err := ma.AssembleValue().AssignBytes(rct.PostState); err != nil {
			return err
		}
		if err := ma.AssembleKey().AssignString("Status"); err != nil {
			return err
		}
		return ma.AssembleValue().AssignNull()
	}

	if err := ma.AssembleKey().AssignString("Status"); err != nil {
		return err
	}
	switch rct.Status {
	case types.ReceiptStatusFailed:
		if err := ma.AssembleValue().AssignBytes(receiptStatusFailedRLP); err != nil {
			return err
		}
	case types.ReceiptStatusSuccessful:
		if err := ma.AssembleValue().AssignBytes(receiptStatusSuccessfulRLP); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized Receipt Status")
	}
	if err := ma.AssembleKey().AssignString("PostState"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignNull()
}

func unpackCumulativeGasUsed(ma ipld.MapAssembler, rct types.Receipt) error {
	cguBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(cguBytes, rct.CumulativeGasUsed)
	if err := ma.AssembleKey().AssignString("CumulativeGasUsed"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(cguBytes)
}

func unpackBloom(ma ipld.MapAssembler, rct types.Receipt) error {
	if err := ma.AssembleKey().AssignString("Bloom"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(rct.Bloom.Bytes())
}

func unpackLogs(ma ipld.MapAssembler, rct types.Receipt) error {
	if err := ma.AssembleKey().AssignString("Logs"); err != nil {
		return err
	}
	la, err := ma.AssembleValue().BeginList(int64(len(rct.Logs)))
	if err != nil {
		return err
	}
	for _, log := range rct.Logs {
		logMa, err := la.AssembleValue().BeginMap(3)
		if err != nil {
			return err
		}
		if err := logMa.AssembleKey().AssignString("Address"); err != nil {
			return err
		}
		if err := logMa.AssembleValue().AssignBytes(log.Address.Bytes()); err != nil {
			return err
		}
		if err := logMa.AssembleKey().AssignString("Data"); err != nil {
			return err
		}
		if err := logMa.AssembleValue().AssignBytes(log.Data); err != nil {
			return err
		}
		if err := logMa.AssembleKey().AssignString("Topics"); err != nil {
			return err
		}
		topicsLa, err := logMa.AssembleValue().BeginList(int64(len(log.Topics)))
		if err != nil {
			return err
		}
		for _, topic := range log.Topics {
			if err := topicsLa.AssembleValue().AssignBytes(topic.Bytes()); err != nil {
				return err
			}
		}
		if err := topicsLa.Finish(); err != nil {
			return err
		}
		if err := logMa.Finish(); err != nil {
			return err
		}
	}
	return la.Finish()
}
