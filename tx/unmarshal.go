package dageth_tx

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ipld/go-ipld-prime"
)

// Decode provides an IPLD codec decode interface for ETH transaction IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x93 when this package is invoked via init.
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
	var tx types.Transaction
	if err := tx.UnmarshalBinary(src); err != nil {
		return err
	}
	ma, err := na.BeginMap(12)
	if err != nil {
		return err
	}
	for _, upFunc := range RequiredUnpackFuncs {
		if err := upFunc(ma, tx); err != nil {
			return fmt.Errorf("invalid DAG-ETH Header binary (%v)", err)
		}
	}
	return ma.Finish()
}

var RequiredUnpackFuncs = []func(ma ipld.MapAssembler, tx types.Transaction) error{
	unpackTxType,
	unpackChainID,
	unpackAccountNonce,
	unpackGasPrice,
	unpackGasLimit,
	unpackRecipient,
	unpackAmount,
	unpackData,
	unpackAccessList,
	unpackSignatureValues,
}

func unpackTxType(ma ipld.MapAssembler, tx types.Transaction) error {
	if err := ma.AssembleKey().AssignString("TxType"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes([]byte{tx.Type()})
}

func unpackChainID(ma ipld.MapAssembler, tx types.Transaction) error {
	// We could make ChainID a required field in the schema even though legacy txs dont include it in the consensus encoding
	if err := ma.AssembleKey().AssignString("ChainID"); err != nil {
		return err
	}
	if tx.Type() == types.LegacyTxType {
		return ma.AssembleValue().AssignNull()
	}
	return ma.AssembleValue().AssignBytes(tx.ChainId().Bytes())
}

func unpackAccountNonce(ma ipld.MapAssembler, tx types.Transaction) error {
	if err := ma.AssembleKey().AssignString("AccountNonce"); err != nil {
		return err
	}
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, tx.Nonce())
	return ma.AssembleValue().AssignBytes(nonceBytes)
}

func unpackGasPrice(ma ipld.MapAssembler, tx types.Transaction) error {
	if err := ma.AssembleKey().AssignString("GasPrice"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(tx.GasPrice().Bytes())
}

func unpackGasLimit(ma ipld.MapAssembler, tx types.Transaction) error {
	if err := ma.AssembleKey().AssignString("GasLimit"); err != nil {
		return err
	}
	gasBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(gasBytes, tx.Gas())
	return ma.AssembleValue().AssignBytes(gasBytes)
}

func unpackRecipient(ma ipld.MapAssembler, tx types.Transaction) error {
	if err := ma.AssembleKey().AssignString("Recipient"); err != nil {
		return err
	}
	if tx.To() == nil {
		return ma.AssembleValue().AssignNull()
	}
	return ma.AssembleValue().AssignBytes(tx.To().Bytes())
}

func unpackAmount(ma ipld.MapAssembler, tx types.Transaction) error {
	if err := ma.AssembleKey().AssignString("Amount"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(tx.Value().Bytes())
}

func unpackData(ma ipld.MapAssembler, tx types.Transaction) error {
	if err := ma.AssembleKey().AssignString("Data"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(tx.Data())
}

func unpackAccessList(ma ipld.MapAssembler, tx types.Transaction) error {
	if err := ma.AssembleKey().AssignString("AccessList"); err != nil {
		return err
	}
	if tx.Type() == types.LegacyTxType {
		return ma.AssembleValue().AssignNull()
	}
	accessList, err := ma.AssembleValue().BeginList(int64(len(tx.AccessList())))
	if err != nil {
		return err
	}
	for _, accessElement := range tx.AccessList() {
		// node := dageth.Type.AccessElement.NewBuilder()
		accessElementMap, err := accessList.AssembleValue().BeginMap(2)
		if err != nil {
			return err
		}
		if err := accessElementMap.AssembleKey().AssignString("Address"); err != nil {
			return err
		}
		if err := accessElementMap.AssembleValue().AssignBytes(accessElement.Address.Bytes()); err != nil {
			return err
		}
		if err := accessElementMap.AssembleKey().AssignString("StorageKeys"); err != nil {
			return err
		}
		storageKeyList, err := accessElementMap.AssembleValue().BeginList(int64(len(accessElement.StorageKeys)))
		if err != nil {
			return err
		}
		for _, storageKey := range accessElement.StorageKeys {
			if err := storageKeyList.AssembleValue().AssignBytes(storageKey.Bytes()); err != nil {
				return err
			}
		}
		if err := storageKeyList.Finish(); err != nil {
			return err
		}
		if err := accessElementMap.Finish(); err != nil {
			return err
		}
	}
	return accessList.Finish()
}

func unpackSignatureValues(ma ipld.MapAssembler, tx types.Transaction) error {
	v, r, s := tx.RawSignatureValues()
	if err := ma.AssembleKey().AssignString("R"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(r.Bytes()); err != nil {
		return err
	}
	if err := ma.AssembleKey().AssignString("S"); err != nil {
		return err
	}
	if err := ma.AssembleValue().AssignBytes(s.Bytes()); err != nil {
		return err
	}
	if err := ma.AssembleKey().AssignString("V"); err != nil {
		return err
	}
	return ma.AssembleValue().AssignBytes(v.Bytes())
}
