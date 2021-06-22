//go:build ignore
// +build ignore

package main

// based on https://github.com/ipld/go-ipld-prime-proto/blob/master/gen/main.go

import (
	"fmt"
	"os"

	"github.com/ipld/go-ipld-prime"
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
	// we could more explicitly type our links with SpawnLinkReference
	ts.Accumulate(schema.SpawnLink("Link"))
	ts.Accumulate(schema.SpawnBytes("Bytes"))
	ts.Accumulate(schema.SpawnString("String"))
	ts.Accumulate(schema.SpawnBytes("BigInt"))
	ts.Accumulate(schema.SpawnBytes("Uint"))
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
			Nonce Uint
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
			schema.SpawnStructField("Nonce", "Uint", false, false),
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

		type Transactions [Transaction]
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
			schema.SpawnStructField("TxType", "TxType", false, false),
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
	ts.Accumulate(schema.SpawnList("Transactions", "Transaction", false))

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
			Status	          Uint   // nullable
			PostState		  Hash   // nullable
			CumulativeGasUsed Uint
			Bloom             Bloom
			Logs              Logs
		}

		type Receipts [Receipt]
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
			schema.SpawnStructField("TxType", "TxType", false, false),
			schema.SpawnStructField("PostState", "Bytes", false, true),
			schema.SpawnStructField("Status", "Uint", false, true),
			schema.SpawnStructField("CumulativeGasUsed", "Uint", false, false),
			schema.SpawnStructField("Bloom", "Bloom", false, false),
			schema.SpawnStructField("Logs", "Logs", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnList("Receipts", "Receipt", false))
}

func accumulateStateDataStructures(ts *schema.TypeSystem) {
	/*
		# TrieNode IPLD
		# Node IPLD values are RLP encoded; node IPLD multihashes are always the KECCAK_256 hash of the RLP encoded node bytes and the codec is dependent on the type of the trie
		type TrieNode union {
			| TrieBranchNode "branch"
			| TrieExtensionNode "extension"
			| TrieLeafNode "leaf"
		} representation keyed


		# The below are the expanded representations for the different types of TrieNodes: branch, extension, and leaf
		type TrieBranchNode struct {
			Child0 nullable Child
			Child1 nullable Child
			Child2 nullable Child
			Child3 nullable Child
			Child4 nullable Child
			Child5 nullable Child
			Child6 nullable Child
			Child7 nullable Child
			Child8 nullable Child
			Child9 nullable Child
			ChildA nullable Child
			ChildB nullable Child
			ChildC nullable Child
			ChildD nullable Child
			ChildE nullable Child
			ChildF nullable Child
			Value  nullable Value
		}

		# Value union type used to handle the different values stored in leaf nodes in the different tries
		type Value union {
			| Transaction "tx"
			| Receipt "rct"
			| Account "state"
			| Bytes "storage"
		} representation keyed

		# Child union type used to handle the case where the node is stored directly in the parent node because it is smaller
		# than the hash that would otherwise reference the node
		type Child union {
			| Link &TrieNode
			| TrieNode TrieNode
		} representation kinded

		type TrieExtensionNode struct {
			PartialPath Bytes
			Child Child
		}

		type TrieLeafNode struct {
			PartialPath Bytes
			Value       Value
		}
	*/
	ts.Accumulate(schema.SpawnUnion("Value",
		[]schema.TypeName{
			"Transaction",
			"Receipt",
			"Account",
			"Bytes",
		},
		schema.SpawnUnionRepresentationKeyed(map[string]schema.TypeName{
			"tx":      "Transaction",
			"rct":     "Receipt",
			"state":   "Account",
			"storage": "Bytes",
		}),
	))
	ts.Accumulate(schema.SpawnUnion("Child",
		[]schema.TypeName{
			"Link",
			"TrieNode",
		},
		schema.SpawnUnionRepresentationKinded(map[ipld.Kind]schema.TypeName{
			ipld.Kind_Link: "Link",
			ipld.Kind_Map:  "TrieNode",
		}),
	))
	ts.Accumulate(schema.SpawnStruct("TrieBranchNode",
		[]schema.StructField{
			schema.SpawnStructField("Child0", "Child", false, true),
			schema.SpawnStructField("Child1", "Child", false, true),
			schema.SpawnStructField("Child2", "Child", false, true),
			schema.SpawnStructField("Child3", "Child", false, true),
			schema.SpawnStructField("Child4", "Child", false, true),
			schema.SpawnStructField("Child5", "Child", false, true),
			schema.SpawnStructField("Child6", "Child", false, true),
			schema.SpawnStructField("Child7", "Child", false, true),
			schema.SpawnStructField("Child8", "Child", false, true),
			schema.SpawnStructField("Child9", "Child", false, true),
			schema.SpawnStructField("ChildA", "Child", false, true),
			schema.SpawnStructField("ChildB", "Child", false, true),
			schema.SpawnStructField("ChildC", "Child", false, true),
			schema.SpawnStructField("ChildD", "Child", false, true),
			schema.SpawnStructField("ChildE", "Child", false, true),
			schema.SpawnStructField("ChildF", "Child", false, true),
			schema.SpawnStructField("Value", "Value", false, true),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnStruct("TrieExtensionNode",
		[]schema.StructField{
			schema.SpawnStructField("PartialPath", "Bytes", false, false),
			schema.SpawnStructField("Child", "Child", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnStruct("TrieLeafNode",
		[]schema.StructField{
			schema.SpawnStructField("PartialPath", "Bytes", false, false),
			schema.SpawnStructField("Value", "Value", false, false),
		},
		schema.SpawnStructRepresentationMap(nil),
	))
	ts.Accumulate(schema.SpawnUnion("TrieNode",
		[]schema.TypeName{
			"TrieBranchNode",
			"TrieExtensionNode",
			"TrieLeafNode",
		},
		schema.SpawnUnionRepresentationKeyed(map[string]schema.TypeName{
			"branch":    "TrieBranchNode",
			"extension": "TrieExtensionNode",
			"leaf":      "TrieLeafNode",
		}),
	))
	/*
		type ByteCode bytes

		type Account struct {
			Nonce    Uint
			Balance  Balance
			StorageRootCID &StorageTrieNode
			CodeCID &ByteCode
		}
	*/
	ts.Accumulate(schema.SpawnBytes("ByteCode"))
	ts.Accumulate(schema.SpawnStruct("Account",
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
