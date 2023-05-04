package plugin

import (
	"github.com/ipfs/kubo/plugin"
	"github.com/ipld/go-ipld-prime/multicodec"

	"github.com/vulcanize/go-codec-dageth/header"
	"github.com/vulcanize/go-codec-dageth/log"
	"github.com/vulcanize/go-codec-dageth/log_trie"
	"github.com/vulcanize/go-codec-dageth/rct"
	"github.com/vulcanize/go-codec-dageth/rct_trie"
	account "github.com/vulcanize/go-codec-dageth/state_account"
	"github.com/vulcanize/go-codec-dageth/state_trie"
	"github.com/vulcanize/go-codec-dageth/storage_trie"
	"github.com/vulcanize/go-codec-dageth/tx"
	"github.com/vulcanize/go-codec-dageth/tx_trie"
	"github.com/vulcanize/go-codec-dageth/uncles"
)

// Plugins is exported list of plugins that will be loaded
var Plugins = []plugin.Plugin{
	&ethIPLDPlugin{},
}

type ethIPLDPlugin struct{}

var _ plugin.PluginIPLD = (*ethIPLDPlugin)(nil)

// Name satisfies the Plugin interface
func (*ethIPLDPlugin) Name() string {
	return "ipld-dag-eth"
}

// Version satisfies the Plugin interface
func (*ethIPLDPlugin) Version() string {
	return "0.0.1"
}

// Init satisfies the Plugin interface
func (*ethIPLDPlugin) Init(_ *plugin.Environment) error {
	return nil
}

// Register satisfies the PluginIPLD interface
func (*ethIPLDPlugin) Register(reg multicodec.Registry) error {
	reg.RegisterDecoder(header.MultiCodecType, header.Decode)
	reg.RegisterDecoder(uncles.MultiCodecType, uncles.Decode)
	reg.RegisterDecoder(tx.MultiCodecType, tx.Decode)
	reg.RegisterDecoder(tx_trie.MultiCodecType, tx_trie.Decode)
	reg.RegisterDecoder(rct.MultiCodecType, rct.Decode)
	reg.RegisterDecoder(rct_trie.MultiCodecType, rct_trie.Decode)
	reg.RegisterDecoder(log.MultiCodecType, log.Decode)
	reg.RegisterDecoder(log_trie.MultiCodecType, log_trie.Decode)
	reg.RegisterDecoder(state_trie.MultiCodecType, state_trie.Decode)
	reg.RegisterDecoder(account.MultiCodecType, account.Decode)
	reg.RegisterDecoder(storage_trie.MultiCodecType, storage_trie.Decode)

	reg.RegisterEncoder(header.MultiCodecType, header.Encode)
	reg.RegisterEncoder(uncles.MultiCodecType, uncles.Encode)
	reg.RegisterEncoder(tx.MultiCodecType, tx.Encode)
	reg.RegisterEncoder(tx_trie.MultiCodecType, tx_trie.Encode)
	reg.RegisterEncoder(rct.MultiCodecType, rct.Encode)
	reg.RegisterEncoder(rct_trie.MultiCodecType, rct_trie.Encode)
	reg.RegisterEncoder(log.MultiCodecType, log.Encode)
	reg.RegisterEncoder(log_trie.MultiCodecType, log_trie.Encode)
	reg.RegisterEncoder(state_trie.MultiCodecType, state_trie.Encode)
	reg.RegisterEncoder(account.MultiCodecType, account.Encode)
	reg.RegisterEncoder(storage_trie.MultiCodecType, storage_trie.Encode)
	return nil
}
