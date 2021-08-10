package log_test

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipld/go-ipld-prime"

	dageth "github.com/vulcanize/go-codec-dageth"
	"github.com/vulcanize/go-codec-dageth/log"
)

var (
	mockLog = &types.Log{
		Address: common.BytesToAddress([]byte{0x11}),
		Topics:  []common.Hash{common.HexToHash("hello"), common.HexToHash("moon"), common.HexToHash("goodbye"), common.HexToHash("world")},
		Data: []byte{0x01, 0x00, 0xff, 0x01, 0x00, 0xff, 0x01, 0x00, 0xff, 0x01, 0x00, 0xff, 0x01, 0x00, 0xff, 0x01,
			0x02, 0x01, 0x00, 0x02, 0x01, 0x00, 0x02, 0x01, 0x00, 0x02, 0x01, 0x00, 0x02, 0x01, 0x00, 0x02,
			0x03, 0x02, 0x01, 0x03, 0x02, 0x01, 0x03, 0x02, 0x01, 0x03, 0x02, 0x01, 0x03, 0x02, 0x01, 0x03},
	}
	logEncoding, _ = rlp.EncodeToBytes(mockLog)
	logNode        ipld.Node
)

/* IPLD Schemas
type Topics [Hash]

type Log struct {
	Address Address
	Topics  Topics
	Data    Bytes
}
*/

func TestLogCodec(t *testing.T) {
	testLogDecoding(t)
	testLogNodeContents(t)
	testLogEncoding(t)
}

func testLogDecoding(t *testing.T) {
	logBuilder := dageth.Type.Log.NewBuilder()
	logReader := bytes.NewReader(logEncoding)
	if err := log.Decode(logBuilder, logReader); err != nil {
		t.Fatalf("unable to decode log into an IPLD node: %v", err)
	}
	logNode = logBuilder.Build()
}

func testLogNodeContents(t *testing.T) {
	addressNode, err := logNode.LookupByString("Address")
	if err != nil {
		t.Fatalf("log is missing Address: %v", err)
	}
	addrBytes, err := addressNode.AsBytes()
	if err != nil {
		t.Fatalf("log Address should be of type Bytes")
	}
	if !bytes.Equal(addrBytes, mockLog.Address.Bytes()) {
		t.Errorf("log Address (%x) does not match expected Address (%x)", addrBytes, mockLog.Address.Bytes())
	}

	dataNode, err := logNode.LookupByString("Data")
	if err != nil {
		t.Fatalf("log is missing Data: %v", err)
	}
	data, err := dataNode.AsBytes()
	if err != nil {
		t.Fatalf("log Data should be of type Bytes")
	}
	if !bytes.Equal(data, mockLog.Data) {
		t.Errorf("log Data (%x) does not match expected data (%x)", data, mockLog.Data)
	}

	topicsNode, err := logNode.LookupByString("Topics")
	if err != nil {
		t.Fatalf("log is missing Topics: %v", err)
	}
	if topicsNode.Length() != 4 {
		t.Fatal("log should have two topics")
	}
	topicsLI := topicsNode.ListIterator()
	for !topicsLI.Done() {
		j, topicNode, err := topicsLI.Next()
		if err != nil {
			t.Fatalf("receipt log topic iterator error: %v", err)
		}
		currentTopic := mockLog.Topics[j].Bytes()
		topicBy, err := topicNode.AsBytes()
		if err != nil {
			t.Fatalf("log Topic should be of type Bytes: %v", err)
		}
		if !bytes.Equal(topicBy, currentTopic) {
			t.Errorf("log topic%d (%x) does not match expected topic%d (%x)", j, topicBy, j, currentTopic)
		}
	}
}

func testLogEncoding(t *testing.T) {
	logWriter := new(bytes.Buffer)
	if err := log.Encode(logNode, logWriter); err != nil {
		t.Fatalf("unable to encode log into writer: %v", err)
	}
	logBytes := logWriter.Bytes()
	if !bytes.Equal(logBytes, logEncoding) {
		t.Errorf("log encoding (%x) does not match the expected consensus encoding (%x)", logBytes, logEncoding)
	}
}
