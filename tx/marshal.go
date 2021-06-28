package tx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/shared"
)

// Encode provides an IPLD codec encode interface for eth transaction IPLDs.
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
	txType, err := shared.GetTxType(node)
	if err != nil {
		return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
	}
	wbs := shared.NewWriteableByteSlice(&enc)
	switch txType {
	case types.LegacyTxType:
		tx, err := packLegacyTx(node)
		if err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
		}
		if err := rlp.Encode(wbs, tx); err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
		}
		return enc, nil
	case types.AccessListTxType:
		tx, err := packAccessListTx(node)
		if err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
		}
		enc = append(enc, txType)
		if err := rlp.Encode(wbs, tx); err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
		}
		return enc, nil
	case types.DynamicFeeTxType:
		tx, err := packDynamicFeeTx(node)
		if err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
		}
		enc = append(enc, txType)
		if err := rlp.Encode(wbs, tx); err != nil {
			return enc, fmt.Errorf("invalid DAG-ETH Transaction form (%v)", err)
		}
		return enc, nil
	default:
		return enc, fmt.Errorf("invalid DAG-ETH Transaction form (unrecognized TxType %d)", txType)
	}
}

// EncodeTx packs the node into a go-ethereum Transaction
func EncodeTx(tx *types.Transaction, inNode ipld.Node) error {
	buf := new(bytes.Buffer)
	if err := Encode(inNode, buf); err != nil {
		return err
	}
	return tx.UnmarshalBinary(buf.Bytes())
}

func packLegacyTx(node ipld.Node) (*types.LegacyTx, error) {
	lTx := &types.LegacyTx{}
	for _, pFunc := range requiredPackLegacyTxFuncs {
		if err := pFunc(lTx, node); err != nil {
			return nil, err
		}
	}
	return lTx, nil
}

func packAccessListTx(node ipld.Node) (*types.AccessListTx, error) {
	alTx := &types.AccessListTx{}
	for _, pFunc := range requiredPackAccessListTxFuncs {
		if err := pFunc(alTx, node); err != nil {
			return nil, err
		}
	}
	return alTx, nil
}

func packDynamicFeeTx(node ipld.Node) (*types.DynamicFeeTx, error) {
	alTx := &types.DynamicFeeTx{}
	for _, pFunc := range requiredPackDynamicFeeTxFuncs {
		if err := pFunc(alTx, node); err != nil {
			return nil, err
		}
	}
	return alTx, nil
}

var requiredPackLegacyTxFuncs = []func(*types.LegacyTx, ipld.Node) error{
	packAccountNonce,
	packGasPrice,
	packGasLimit,
	packRecipient,
	packAmount,
	packData,
	packSignatureValues,
}

var requiredPackAccessListTxFuncs = []func(*types.AccessListTx, ipld.Node) error{
	packChainIDAL,
	packAccountNonceAL,
	packGasPriceAL,
	packGasLimitAL,
	packRecipientAL,
	packAmountAL,
	packDataAL,
	packAccessListAL,
	packSignatureValuesAL,
}

var requiredPackDynamicFeeTxFuncs = []func(*types.DynamicFeeTx, ipld.Node) error{
	packChainIDDF,
	packAccountNonceDF,
	packGasTipCap,
	packGasFeeCap,
	packGasLimitDF,
	packRecipientDF,
	packAmountDF,
	packDataDF,
	packAccessListDF,
	packSignatureValuesDF,
}

func packChainIDAL(tx *types.AccessListTx, node ipld.Node) error {
	chainIDNode, err := node.LookupByString("ChainID")
	if err != nil {
		return err
	}
	chainIDBytes, err := chainIDNode.AsBytes()
	if err != nil {
		return err
	}
	tx.ChainID = new(big.Int).SetBytes(chainIDBytes)
	return nil
}

func packChainIDDF(tx *types.DynamicFeeTx, node ipld.Node) error {
	chainIDNode, err := node.LookupByString("ChainID")
	if err != nil {
		return err
	}
	chainIDBytes, err := chainIDNode.AsBytes()
	if err != nil {
		return err
	}
	tx.ChainID = new(big.Int).SetBytes(chainIDBytes)
	return nil
}

func packAccountNonce(tx *types.LegacyTx, node ipld.Node) error {
	nonceNode, err := node.LookupByString("AccountNonce")
	if err != nil {
		return err
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		return err
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	tx.Nonce = nonce
	return nil
}

func packAccountNonceAL(tx *types.AccessListTx, node ipld.Node) error {
	nonceNode, err := node.LookupByString("AccountNonce")
	if err != nil {
		return err
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		return err
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	tx.Nonce = nonce
	return nil
}

func packAccountNonceDF(tx *types.DynamicFeeTx, node ipld.Node) error {
	nonceNode, err := node.LookupByString("AccountNonce")
	if err != nil {
		return err
	}
	nonceBytes, err := nonceNode.AsBytes()
	if err != nil {
		return err
	}
	nonce := binary.BigEndian.Uint64(nonceBytes)
	tx.Nonce = nonce
	return nil
}

func packGasPrice(tx *types.LegacyTx, node ipld.Node) error {
	gpNode, err := node.LookupByString("GasPrice")
	if err != nil {
		return err
	}
	gpBytes, err := gpNode.AsBytes()
	if err != nil {
		return err
	}
	gp := new(big.Int).SetBytes(gpBytes)
	tx.GasPrice = gp
	return nil
}

func packGasPriceAL(tx *types.AccessListTx, node ipld.Node) error {
	gpNode, err := node.LookupByString("GasPrice")
	if err != nil {
		return err
	}
	gpBytes, err := gpNode.AsBytes()
	if err != nil {
		return err
	}
	gp := new(big.Int).SetBytes(gpBytes)
	tx.GasPrice = gp
	return nil
}

func packGasTipCap(tx *types.DynamicFeeTx, node ipld.Node) error {
	gtcNode, err := node.LookupByString("GasTipCap")
	if err != nil {
		return err
	}
	gtcBytes, err := gtcNode.AsBytes()
	if err != nil {
		return err
	}
	gtc := new(big.Int).SetBytes(gtcBytes)
	tx.GasTipCap = gtc
	return nil
}

func packGasFeeCap(tx *types.DynamicFeeTx, node ipld.Node) error {
	gfcNode, err := node.LookupByString("GasFeeCap")
	if err != nil {
		return err
	}
	gfcBytes, err := gfcNode.AsBytes()
	if err != nil {
		return err
	}
	gfc := new(big.Int).SetBytes(gfcBytes)
	tx.GasFeeCap = gfc
	return nil
}

func packGasLimit(tx *types.LegacyTx, node ipld.Node) error {
	glNode, err := node.LookupByString("GasLimit")
	if err != nil {
		return err
	}
	glBytes, err := glNode.AsBytes()
	if err != nil {
		return err
	}
	gl := binary.BigEndian.Uint64(glBytes)
	tx.Gas = gl
	return nil
}

func packGasLimitAL(tx *types.AccessListTx, node ipld.Node) error {
	glNode, err := node.LookupByString("GasLimit")
	if err != nil {
		return err
	}
	glBytes, err := glNode.AsBytes()
	if err != nil {
		return err
	}
	gl := binary.BigEndian.Uint64(glBytes)
	tx.Gas = gl
	return nil
}

func packGasLimitDF(tx *types.DynamicFeeTx, node ipld.Node) error {
	glNode, err := node.LookupByString("GasLimit")
	if err != nil {
		return err
	}
	glBytes, err := glNode.AsBytes()
	if err != nil {
		return err
	}
	gl := binary.BigEndian.Uint64(glBytes)
	tx.Gas = gl
	return nil
}

func packRecipient(tx *types.LegacyTx, node ipld.Node) error {
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
	tx.To = &recipient
	return nil
}

func packRecipientAL(tx *types.AccessListTx, node ipld.Node) error {
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
	tx.To = &recipient
	return nil
}

func packRecipientDF(tx *types.DynamicFeeTx, node ipld.Node) error {
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
	tx.To = &recipient
	return nil
}

func packAmount(tx *types.LegacyTx, node ipld.Node) error {
	aNode, err := node.LookupByString("Amount")
	if err != nil {
		return err
	}
	aBytes, err := aNode.AsBytes()
	if err != nil {
		return err
	}
	amount := new(big.Int).SetBytes(aBytes)
	tx.Value = amount
	return nil
}

func packAmountAL(tx *types.AccessListTx, node ipld.Node) error {
	aNode, err := node.LookupByString("Amount")
	if err != nil {
		return err
	}
	aBytes, err := aNode.AsBytes()
	if err != nil {
		return err
	}
	amount := new(big.Int).SetBytes(aBytes)
	tx.Value = amount
	return nil
}

func packAmountDF(tx *types.DynamicFeeTx, node ipld.Node) error {
	aNode, err := node.LookupByString("Amount")
	if err != nil {
		return err
	}
	aBytes, err := aNode.AsBytes()
	if err != nil {
		return err
	}
	amount := new(big.Int).SetBytes(aBytes)
	tx.Value = amount
	return nil
}

func packData(tx *types.LegacyTx, node ipld.Node) error {
	dNode, err := node.LookupByString("Data")
	if err != nil {
		return err
	}
	dBytes, err := dNode.AsBytes()
	if err != nil {
		return err
	}
	tx.Data = dBytes
	return nil
}

func packDataAL(tx *types.AccessListTx, node ipld.Node) error {
	dNode, err := node.LookupByString("Data")
	if err != nil {
		return err
	}
	dBytes, err := dNode.AsBytes()
	if err != nil {
		return err
	}
	tx.Data = dBytes
	return nil
}

func packDataDF(tx *types.DynamicFeeTx, node ipld.Node) error {
	dNode, err := node.LookupByString("Data")
	if err != nil {
		return err
	}
	dBytes, err := dNode.AsBytes()
	if err != nil {
		return err
	}
	tx.Data = dBytes
	return nil
}

func packAccessListAL(tx *types.AccessListTx, node ipld.Node) error {
	accessList, err := createAccessList(node)
	if err != nil {
		return err
	}
	tx.AccessList = accessList
	return nil
}

func packAccessListDF(tx *types.DynamicFeeTx, node ipld.Node) error {
	accessList, err := createAccessList(node)
	if err != nil {
		return err
	}
	tx.AccessList = accessList
	return nil
}

func createAccessList(node ipld.Node) (types.AccessList, error) {
	alNode, err := node.LookupByString("AccessList")
	if err != nil {
		return nil, err
	}
	accessList := make(types.AccessList, alNode.Length())
	accessListIt := alNode.ListIterator()
	for !accessListIt.Done() {
		index, accessElementNode, err := accessListIt.Next()
		if err != nil {
			return nil, err
		}
		addrNode, err := accessElementNode.LookupByString("Address")
		if err != nil {
			return nil, err
		}
		addrBytes, err := addrNode.AsBytes()
		if err != nil {
			return nil, err
		}
		addr := common.BytesToAddress(addrBytes)

		storageKeysNode, err := accessElementNode.LookupByString("StorageKeys")
		if err != nil {
			return nil, err
		}
		storageKeys := make([]common.Hash, storageKeysNode.Length())
		storageKeysIt := storageKeysNode.ListIterator()
		for !storageKeysIt.Done() {
			index, storageKeyNode, err := storageKeysIt.Next()
			if err != nil {
				return nil, err
			}
			storageKeyBytes, err := storageKeyNode.AsBytes()
			if err != nil {
				return nil, err
			}
			storageKeys[index] = common.BytesToHash(storageKeyBytes)
		}
		accessElement := types.AccessTuple{
			Address:     addr,
			StorageKeys: storageKeys,
		}
		accessList[index] = accessElement
	}
	return accessList, nil
}

func createVRS(node ipld.Node) (*big.Int, *big.Int, *big.Int, error) {
	vNode, err := node.LookupByString("V")
	if err != nil {
		return nil, nil, nil, err
	}
	vBytes, err := vNode.AsBytes()
	if err != nil {
		return nil, nil, nil, err
	}
	v := new(big.Int).SetBytes(vBytes)
	rNode, err := node.LookupByString("R")
	if err != nil {
		return nil, nil, nil, err
	}
	rBytes, err := rNode.AsBytes()
	if err != nil {
		return nil, nil, nil, err
	}
	r := new(big.Int).SetBytes(rBytes)
	sNode, err := node.LookupByString("S")
	if err != nil {
		return nil, nil, nil, err
	}
	sBytes, err := sNode.AsBytes()
	if err != nil {
		return nil, nil, nil, err
	}
	s := new(big.Int).SetBytes(sBytes)
	return v, r, s, nil
}

func packSignatureValues(tx *types.LegacyTx, node ipld.Node) error {
	v, r, s, err := createVRS(node)
	if err != nil {
		return err
	}
	tx.V = v
	tx.R = r
	tx.S = s
	return nil
}

func packSignatureValuesAL(tx *types.AccessListTx, node ipld.Node) error {
	v, r, s, err := createVRS(node)
	if err != nil {
		return err
	}
	tx.V = v
	tx.R = r
	tx.S = s
	return nil
}

func packSignatureValuesDF(tx *types.DynamicFeeTx, node ipld.Node) error {
	v, r, s, err := createVRS(node)
	if err != nil {
		return err
	}
	tx.V = v
	tx.R = r
	tx.S = s
	return nil
}
