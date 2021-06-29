/*
Package dageth provides a Go implementation of the IPLD DAG-ETH spec
(https://github.com/ipld/ipld/tree/master/specs/codecs/ethereum) for
go-ipld-prime (https://github.com/ipld/go-ipld-prime/).

Use the Decode() and Encode() functions directly, or import one of the packages to have their codec
registered into the go-ipld-prime multicodec registry and available from the
cidlink.DefaultLinkSystem.

Nodes encoded with theses codecs _must_ conform to the DAG-ETH spec. Specifically,
they should have the non-optional fields shown in the DAG-ETH [schemas](https://github.com/ipld/ipld/tree/master/specs/codecs/ethereum):

Use the dageth.Type slab to select the appropriate type (e.g. dageth.Type.Transaction) for strictness guarantees.
Basic ipld.Nodes will need to have the appropriate fields (and no others) to successfully encode using this codec.
*/
package dageth

//go:generate go run gen.go
//go:generate gofmt -w ipldsch_minima.go ipldsch_satisfaction.go ipldsch_types.go