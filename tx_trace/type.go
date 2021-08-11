package tx_trace

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// TxTrace contains the EVM context, input, and output for each OPCODE in a transaction that was applied to a specific state
type TxTrace struct {
	TxHashes  []common.Hash
	StateRoot common.Hash
	Result    []byte
	Frames    []Frame
	Gas       uint64
	Failed    bool
}

// Frame represents the EVM context, input, and output for a specific OPCODE during a transaction trace
type Frame struct {
	Op     vm.OpCode
	From   common.Address
	To     common.Address
	Input  []byte
	Output []byte
	Gas    uint64
	Cost   uint64
	Value  *big.Int
}
