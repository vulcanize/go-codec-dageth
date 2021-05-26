//go:build ignore
// +build ignore

package main

// based on https://github.com/ipld/go-ipld-prime-proto/blob/master/gen/main.go

import (
	"fmt"
	"os"

	"github.com/ipld/go-ipld-prime/schema"
	gengo "github.com/ipld/go-ipld-prime/schema/gen/go"
)

const (
	pkgName = "dageth"
)

func main() {
	// initialize a new type system
	ts := new(schema.TypeSystem)
	ts.Init()

	// accumulate the different types
	accumulateBasicTypes(ts)
	accumulateChainTypes(ts)
	accumulateConvenienceTypes(ts)
	accumulateStateDataStructures(ts)

	// verify internal correctness of the types
	if errs := ts.ValidateGraph(); errs != nil {
		for _, err := range errs {
			fmt.Printf("- %s\n", err)
		}
		os.Exit(1)
	}

	// generate the code
	adjCfg := &gengo.AdjunctCfg{}
	gengo.Generate(".", pkgName, *ts, adjCfg)
}

func accumulateBasicTypes(ts *schema.TypeSystem) {
	ts.Accumulate(schema.SpawnString("String"))
	ts.Accumulate(schema.SpawnInt("Int"))
	// we could more explicitly type our links with SpawnLinkReference
	ts.Accumulate(schema.SpawnLink("Link"))
	ts.Accumulate(schema.SpawnBytes("Bytes"))

	ts.Accumulate(schema.SpawnBytes("BigInt"))
	ts.Accumulate(schema.SpawnBytes("Uint"))
	ts.Accumulate(schema.SpawnBytes("BlockNonce"))
	ts.Accumulate(schema.SpawnBytes("Hash"))
	ts.Accumulate(schema.SpawnBytes("Address"))
	ts.Accumulate(schema.SpawnBytes("Bloom"))
	ts.Accumulate(schema.SpawnBytes("Balance"))
	ts.Accumulate(schema.SpawnBytes("OpCode"))
	ts.Accumulate(schema.SpawnBytes("Time"))
	ts.Accumulate(schema.SpawnBytes("TxType"))
}

func accumulateChainTypes(ts *schema.TypeSystem) {
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
		    Time Time
		    Extra Bytes
		    MixDigest Hash
		    Nonce BlockNonce
		}
	*/
	ts.Accumulate(schema.SpawnStruct("Header",
		[]schema.StructField{
			schema.SpawnStructField("ParentCID", "Link", false, false),
			schema.SpawnStructField("UnclesCID", "Link", false, false),
			schema.SpawnStructField("Coinbase", "Address", false, false),
			schema.SpawnStructField("StateRootCID", "Link", false, false),
			schema.SpawnStructField("TxRootCID", "Link", false, false),
			schema.SpawnStructField("RctRootCID", "Link", false, false),
			schema.SpawnStructField("Bloom", "Bloom", false, false),
			schema.SpawnStructField("Difficulty", "BigInt", false, false),
			schema.SpawnStructField("Number", "BigInt", false, false),
			schema.SpawnStructField("GasLimit", "Uint", false, false),
			schema.SpawnStructField("GasUsed", "Uint", false, false),
			schema.SpawnStructField("Time", "Time", false, false),
			schema.SpawnStructField("Extra", "Bytes", false, false),
			schema.SpawnStructField("MixDigest", "Hash", false, false),
			schema.SpawnStructField("Nonce", "BlockNonce", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))

	/*
		type Uncles [Header]
	*/
	ts.Accumulate(schema.SpawnList("Uncles", "Header", false))

	/*
		type StorageKeys [Hash]

		type AccessElement struct {
		    Address     Address
		    StorageKeys StorageKeys
		}

		type AccessList [AccessElement]

		type Transaction struct {
			Type         TxType
			// We could make ChainID a required field in the IPLD schema
			ChainID      nullable BigInt # null unless the transaction is an EIP-2930 transaction
			AccountNonce Uint
			GasPrice     BigInt
			GasLimit     Uint
			Recipient    nullable Address # null recipient means the tx is a contract creation
			Amount       BigInt
			Data         Bytes
			AccessList   nullable AccessList # null unless the transaction is an EIP-2930 transaction

			# Signature values
			V            BigInt
			R            BigInt
			S            BigInt
		}
	*/
	ts.Accumulate(schema.SpawnList("StorageKeys", "Hash", false))
	ts.Accumulate(schema.SpawnStruct("AccessElement",
		[]schema.StructField{
			schema.SpawnStructField("Address", "Address", false, false),
			schema.SpawnStructField("StorageKeys", "StorageKeys", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnList("AccessList", "AccessElement", false))
	ts.Accumulate(schema.SpawnStruct("Transaction",
		[]schema.StructField{
			schema.SpawnStructField("Type", "TxType", false, false),
			schema.SpawnStructField("ChainID", "BigInt", false, true),
			schema.SpawnStructField("AccountNonce", "Uint", false, false),
			schema.SpawnStructField("GasPrice", "BigInt", false, false),
			schema.SpawnStructField("GasLimit", "Uint", false, false),
			schema.SpawnStructField("Recipient", "Address", false, true),
			schema.SpawnStructField("Amount", "BigInt", false, false),
			schema.SpawnStructField("Data", "Bytes", false, false),
			schema.SpawnStructField("AccessList", "AccessList", false, true),
			schema.SpawnStructField("V", "BigInt", false, false),
			schema.SpawnStructField("R", "BigInt", false, false),
			schema.SpawnStructField("S", "BigInt", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	/*
		type Topics [Hash]

		type Log struct {
			Address Address
			Topics  Topics
			Data    Bytes
		}

		type Logs [Log]

		type Receipt struct {
			Type			  TxType
			// We could make Status an enum
			Status	          Uint // nullable
			PostState		  Hash   // nullable
			CumulativeGasUsed Uint
			Bloom             Bloom
			Logs              Logs
		}
	*/
	ts.Accumulate(schema.SpawnList("Topics", "Hash", false))
	ts.Accumulate(schema.SpawnStruct("Log",
		[]schema.StructField{
			schema.SpawnStructField("Address", "Address", false, false),
			schema.SpawnStructField("Topics", "Topics", false, false),
			schema.SpawnStructField("Data", "Bytes", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnList("Logs", "Log", false))
	ts.Accumulate(schema.SpawnStruct("Receipt",
		[]schema.StructField{
			schema.SpawnStructField("Type", "TxType", false, false),
			schema.SpawnStructField("PostState", "Bytes", false, true),
			schema.SpawnStructField("Status", "Uint", false, true),
			schema.SpawnStructField("CumulativeGasUsed", "Uint", false, false),
			schema.SpawnStructField("Bloom", "Bloom", false, false),
			schema.SpawnStructField("Logs", "Logs", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
}

func accumulateStateDataStructures(ts *schema.TypeSystem) {
	/*
		type TrieBranchNode struct {
			Child0 nullable &TrieNode
			Child1 nullable &TrieNode
			Child2 nullable &TrieNode
			Child3 nullable &TrieNode
			Child4 nullable &TrieNode
			Child5 nullable &TrieNode
			Child6 nullable &TrieNode
			Child7 nullable &TrieNode
			Child8 nullable &TrieNode
			Child9 nullable &TrieNode
			ChildA nullable &TrieNode
			ChildB nullable &TrieNode
			ChildC nullable &TrieNode
			ChildD nullable &TrieNode
			ChildE nullable &TrieNode
			ChildF nullable &TrieNode
			Value  Bytes
		}

		type TrieExtensionNode struct {
			PartialPath Bytes
			ChildNode   &TrieNode
		}

		type TrieLeafNode struct {
			PartialPath Bytes
			Value       Bytes
		}

		type TrieValueNode struct {
			Value Bytes
		}

		type TrieNode union {
			| TrieBranchNode "branch"
			| TrieExtensionNode "extension"
			| TrieLeafNode "leaf"
			| TrieValueNode "value"
		} representation keyed
	*/
	// This would probably be better represented using the the inline Union but we need a new SpawnUnionRepresentationInline function
	ts.Accumulate(schema.SpawnStruct("TrieBranchNode",
		[]schema.StructField{
			schema.SpawnStructField("Child0", "Link", false, true),
			schema.SpawnStructField("Child1", "Link", false, true),
			schema.SpawnStructField("Child2", "Link", false, true),
			schema.SpawnStructField("Child3", "Link", false, true),
			schema.SpawnStructField("Child4", "Link", false, true),
			schema.SpawnStructField("Child5", "Link", false, true),
			schema.SpawnStructField("Child6", "Link", false, true),
			schema.SpawnStructField("Child7", "Link", false, true),
			schema.SpawnStructField("Child8", "Link", false, true),
			schema.SpawnStructField("Child9", "Link", false, true),
			schema.SpawnStructField("ChildA", "Link", false, true),
			schema.SpawnStructField("ChildB", "Link", false, true),
			schema.SpawnStructField("ChildC", "Link", false, true),
			schema.SpawnStructField("ChildD", "Link", false, true),
			schema.SpawnStructField("ChildE", "Link", false, true),
			schema.SpawnStructField("ChildF", "Link", false, true),
			schema.SpawnStructField("Value", "Bytes", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnStruct("TrieExtensionNode",
		[]schema.StructField{
			schema.SpawnStructField("PartialPath", "Bytes", false, false),
			schema.SpawnStructField("ChildNode", "Link", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnStruct("TrieLeafNode",
		[]schema.StructField{
			schema.SpawnStructField("PartialPath", "Bytes", false, false),
			schema.SpawnStructField("Value", "Bytes", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnStruct("TrieValueNode",
		[]schema.StructField{
			schema.SpawnStructField("Value", "Bytes", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnUnion("TrieNode",
		[]schema.TypeName{
			"TrieBranchNode",
			"TrieExtensionNode",
			"TrieLeafNode",
			"TrieValueNode",
		},
		schema.SpawnUnionRepresentationKeyed(map[string]schema.TypeName{
			"branch":    "TrieBranchNode",
			"extension": "TrieExtensionNode",
			"leaf":      "TrieLeafNode",
			"value":     "TrieValueNode",
		}),
	))

	/*
		type ByteCode bytes

		type StateAccount struct {
			Nonce    Uint
			Balance  Balance
			StorageRootCID &StorageTrieNode
			CodeCID &ByteCode
		}
	*/
	ts.Accumulate(schema.SpawnBytes("ByteCode"))
	ts.Accumulate(schema.SpawnStruct("StateAccount",
		[]schema.StructField{
			schema.SpawnStructField("Nonce", "Uint", false, false),
			schema.SpawnStructField("Balance", "Balance", false, false),
			schema.SpawnStructField("StorageRootCID", "Link", false, false),
			schema.SpawnStructField("CodeCID", "Link", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
}

func accumulateConvenienceTypes(ts *schema.TypeSystem) {
	// TODO: write convenience types
}
