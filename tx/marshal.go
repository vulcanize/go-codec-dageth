package dageth_tx

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
)

// Encode provides an IPLD codec encode interface for eth header IPLDs.
// This function is registered via the go-ipld-prime link loader for multicodec
// code 0x93 when this package is invoked via init.
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
	// Wrap in a typed node for some basic schema form checking
	builder := dageth.Type.Transaction.NewBuilder()
	if err := builder.AssignNode(inNode); err != nil {
		return enc, err
	}
	node := builder.Build()
	txType, err := getType(node)
	if err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
	}
	var tx *types.Transaction
	switch txType {
	case types.LegacyTxType:
		tx, err = packLegacyTx(node)
		if err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
		}
	case types.AccessListTxType:
		tx, err = packAccessListTx(node)
		if err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
		}
	default:
		return enc, fmt.Errorf("invalid DAG-ETH Transaction form (unrecognized TxType %d)", txType)
	}
	encodedTx, err := tx.MarshalBinary()
	if err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Transaction form (unable to binary marshal transaction: %v)", err)
	}
	enc = encodedTx
	return enc, nil
}

func packLegacyTx(node ipld.Node) (*types.Transaction, error) {
	lTx := &types.LegacyTx{}
	for _, pFunc := range RequiredPackFuncs {
		if err := pFunc(lTx, node); err != nil {
			return nil, err
		}
	}
	return types.NewTx(lTx), nil
}

func packAccessListTx(node ipld.Node) (*types.Transaction, error) {
	alTx := &types.AccessListTx{}
	for _, pFunc := range RequiredPackFuncs {
		if err := pFunc(alTx, node); err != nil {
			return nil, err
		}
	}
	return types.NewTx(alTx), nil
}

var RequiredPackFuncs = []func(interface{}, ipld.Node) error{
	packChainID,
	packAccountNonce,
	packGasPrice,
	packGasLimit,
	packRecipient,
	packAmount,
	packData,
	packAccessList,
	packSignatureValues,
}

func getType(node ipld.Node) (uint8, error) {
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

func packChainID(tx interface{}, node ipld.Node) error {
	chainIDNode, err := node.LookupByString("ChainID")
	if err != nil {
		return err
	}
	if chainIDNode.IsNull() { // Throw error if null for accessList tx or if not null for legacy
		return nil
	}
	chainIDBytes, err := chainIDNode.AsBytes()
	if err != nil {
		return err
	}
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.ChainID = new(big.Int).SetBytes(chainIDBytes)
	case *types.LegacyTx:
		return nil
	default:
		return fmt.Errorf("unrecognized tx type")
	}
	return nil
}

func packAccountNonce(tx interface{}, node ipld.Node) error {
	nonceNode, err := node.LookupByString("AccountNonce")
	if err != nil {
		return err
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		return err
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.Nonce = nonce
	case *types.LegacyTx:
		t.Nonce = nonce
	default:
		return fmt.Errorf("unrecognized tx type %T", t)
	}
	return nil
}

func packGasPrice(tx interface{}, node ipld.Node) error {
	gpNode, err := node.LookupByString("GasPrice")
	if err != nil {
		return err
	}
	gpBytes, err := gpNode.AsBytes()
	if err != nil {
		return err
	}
	gp := new(big.Int).SetBytes(gpBytes)
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.GasPrice = gp
	case *types.LegacyTx:
		t.GasPrice = gp
	default:
		return fmt.Errorf("unrecognized tx type %T", t)
	}
	return nil
}

func packGasLimit(tx interface{}, node ipld.Node) error {
	glNode, err := node.LookupByString("GasLimit")
	if err != nil {
		return err
	}
	glBytes, err := glNode.AsBytes()
	if err != nil {
		return err
	}
	gl := binary.BigEndian.Uint64(glBytes)
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.Gas = gl
	case *types.LegacyTx:
		t.Gas = gl
	default:
		return fmt.Errorf("unrecognized tx type %T", t)
	}
	return nil
}

func packRecipient(tx interface{}, node ipld.Node) error {
	rNode, err := node.LookupByString("Recipient")
	if err != nil {
		return err
	}
	if rNode.IsNull() {
		return nil
	}
	rBytes, err := rNode.AsBytes()
	if err != nil {
		return err
	}
	recipient := common.BytesToAddress(rBytes)
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.To = &recipient
	case *types.LegacyTx:
		t.To = &recipient
	default:
		return fmt.Errorf("unrecognized tx type %T", t)
	}
	return nil
}

func packAmount(tx interface{}, node ipld.Node) error {
	aNode, err := node.LookupByString("Amount")
	if err != nil {
		return err
	}
	aBytes, err := aNode.AsBytes()
	if err != nil {
		return err
	}
	amount := new(big.Int).SetBytes(aBytes)
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.Value = amount
	case *types.LegacyTx:
		t.Value = amount
	default:
		return fmt.Errorf("unrecognized tx type %T", t)
	}
	return nil
}

func packData(tx interface{}, node ipld.Node) error {
	dNode, err := node.LookupByString("Data")
	if err != nil {
		return err
	}
	dBytes, err := dNode.AsBytes()
	if err != nil {
		return err
	}
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.Data = dBytes
	case *types.LegacyTx:
		t.Data = dBytes
	default:
		return fmt.Errorf("unrecognized tx type %T", t)
	}
	return nil
}

func packAccessList(tx interface{}, node ipld.Node) error {
	alNode, err := node.LookupByString("AccessList")
	if err != nil {
		return err
	}
	if alNode.IsNull() { // Throw error if null for accessList tx or if not null for legacy
		return nil
	}
	accessList := make(types.AccessList, alNode.Length())
	accessListIt := alNode.ListIterator()
	for !accessListIt.Done() {
		index, accessElementNode, err := accessListIt.Next()
		if err != nil {
			return err
		}
		addrNode, err := accessElementNode.LookupByString("Address")
		if err != nil {
			return err
		}
		addrBytes, err := addrNode.AsBytes()
		if err != nil {
			return err
		}
		addr := common.BytesToAddress(addrBytes)

		storageKeysNode, err := accessElementNode.LookupByString("StorageKeys")
		if err != nil {
			return err
		}
		storageKeys := make([]common.Hash, storageKeysNode.Length())
		storageKeysIt := storageKeysNode.ListIterator()
		for !storageKeysIt.Done() {
			index, storageKeyNode, err := storageKeysIt.Next()
			if err != nil {
				return err
			}
			storageKeyBytes, err := storageKeyNode.AsBytes()
			if err != nil {
				return err
			}
			storageKeys[index] = common.BytesToHash(storageKeyBytes)
		}
		accessElement := types.AccessTuple{
			Address:     addr,
			StorageKeys: storageKeys,
		}
		accessList[index] = accessElement
	}
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.AccessList = accessList
	case *types.LegacyTx:
		return nil
	default:
		return fmt.Errorf("unrecognized tx type %T", t)
	}
	return nil
}

func packSignatureValues(tx interface{}, node ipld.Node) error {
	vNode, err := node.LookupByString("V")
	if err != nil {
		return err
	}
	vBytes, err := vNode.AsBytes()
	if err != nil {
		return err
	}
	v := new(big.Int).SetBytes(vBytes)
	rNode, err := node.LookupByString("R")
	if err != nil {
		return err
	}
	rBytes, err := rNode.AsBytes()
	if err != nil {
		return err
	}
	r := new(big.Int).SetBytes(rBytes)
	sNode, err := node.LookupByString("S")
	if err != nil {
		return err
	}
	sBytes, err := sNode.AsBytes()
	if err != nil {
		return err
	}
	s := new(big.Int).SetBytes(sBytes)
	switch t := tx.(type) {
	case *types.AccessListTx:
		t.V = v
		t.R = r
		t.S = s
	case *types.LegacyTx:
		t.V = v
		t.R = r
		t.S = s
	default:
		return fmt.Errorf("unrecognized tx type %T", t)
	}
	return nil
}
