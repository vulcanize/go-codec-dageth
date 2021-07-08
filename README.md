# go-codec-dageth
A Go implementation of the DAG interface for [Ethereum IPLD types](https://github.com/ipld/ipld/tree/master/specs/codecs/dag-eth) for use with for [go-ipld-prime](https://github.com/ipld/go-ipld-prime/)

Use `Decode(ipld.NodeAssembler, io.Reader)` and `Encode(ipld.Node, io.Writer)` directly, or import the packages to have the codecs registered into the go-ipld-prime CID link loader.

Use the `dageth.Type` slab to select the appropriate type (e.g. `dageth.Type.Transaction`) for strictness guarantees.
Basic `ipld.Node`s will need to have the appropriate fields (and no others) to successfully encode using this codec.

## Supported types
[Header](./header) - 0x90  
[Uncles](./uncles) (Header list) - 0x91  
[Transaction](./tx) - 0x93  
[Transaction Trie Node](./tx_trie) - 0x92  
[Receipt](./rct) - 0x95  
[Receipt Trie Node](./rct_trie) - 0x94  
[State Trie Node](./state_trie) - 0x96  
[State Account](./state_account) - 0x97  
[Storage Trie Node](./storage_trie) - 0x98  

## License & Copyright

Copyright &copy; 2021 Vulcanize Inc

Licensed under either of

* Apache 2.0, ([LICENSE-APACHE](LICENSE-APACHE) / http://www.apache.org/licenses/LICENSE-2.0)
* MIT ([LICENSE-MIT](LICENSE-MIT) / http://opensource.org/licenses/MIT)

### Contribution

Unless you explicitly state otherwise, any contribution intentionally submitted for inclusion in the work by you, as defined in the Apache-2.0 license, shall be dual licensed as above, without any additional terms or conditions.
