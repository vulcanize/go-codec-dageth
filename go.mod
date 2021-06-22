module github.com/vulcanize/go-codec-dageth

go 1.15

require (
	github.com/ethereum/go-ethereum v1.10.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs-blockstore v1.0.4
	github.com/ipfs/go-ipfs-ds-help v1.0.0
	github.com/ipld/go-ipld-prime v0.9.0
	github.com/multiformats/go-multihash v0.0.15
)

replace github.com/ethereum/go-ethereum v1.10.4 => github.com/vulcanize/go-ethereum v1.10.4-ir-0.0.1
