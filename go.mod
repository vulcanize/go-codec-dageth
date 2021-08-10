module github.com/vulcanize/go-codec-dageth

go 1.15

require (
	github.com/ethereum/go-ethereum v1.10.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipld/go-ipld-prime v0.10.0
	github.com/multiformats/go-multihash v0.0.15
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

replace github.com/ethereum/go-ethereum v1.10.4 => github.com/vulcanize/go-ethereum v1.10.4-ir-0.0.1
