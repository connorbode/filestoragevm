// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package filestoragevm

import (
	"errors"
	"time"
	"strconv"

	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/vms/components/core"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting"
)

var (
	errTimestampTooEarly = errors.New("block's timestamp is earlier than its parent's timestamp")
	errDatabaseGet       = errors.New("error while retrieving data from database")
	errDatabaseSave      = errors.New("error while saving block to the database")
	errTimestampTooLate  = errors.New("block's timestamp is more than 1 hour ahead of local time")
	errBlockType         = errors.New("unexpected block type")
	errInvalidSignature  = errors.New("invalid signature")

	_ snowman.Block = &Block{}
)

// Block is a block on the chain.
// Each block contains:
// 1) A piece of data (a string)
// 2) A timestamp
type Block struct {
	*core.Block `serialize:"true"`
	Data        [dataLen]byte `serialize:"true"`
}

// Verify returns nil iff this block is valid.
// To be valid, it must be that:
// b.parent.Timestamp < b.Timestamp <= [local time] + 1 hour
func (b *Block) Verify() error {
	// Check to see if this block has already been verified by calling Verify on the
	// embedded *core.Block.
	// If there is an error while checking, return an error.
	// If the core.Block says the block is accepted, return accepted.
	if accepted, err := b.Block.Verify(); err != nil || accepted {
		return err
	}

	// Get [b]'s parent
	parentID := b.Parent()
	parentIntf, err := b.VM.GetBlock(parentID)
	if err != nil {
		return errDatabaseGet
	}
	parent, ok := parentIntf.(*Block)
	if !ok {
		return errBlockType
	}

	// Ensure [b]'s timestamp is after its parent's timestamp.
	if b.Timestamp().Unix() < parent.Timestamp().Unix() {
		return errTimestampTooEarly
	}

	// Ensure [b]'s timestamp is not more than an hour
	// ahead of this node's time
	if b.Timestamp().Unix()>= time.Now().Add(time.Hour).Unix() {
		return errTimestampTooLate
	}

	// check signatures on the block
	factory := crypto.FactorySECP256K1R{}
	pubkeyBytes := b.Data[0:50]
	pubkeyDecoded, _ := formatting.Decode(formatting.CB58, string(pubkeyBytes))
	pubkey, _ := factory.ToPublicKey(pubkeyDecoded)
	sigLenBytes := b.Data[50:53]
	sigLenStr := string(sigLenBytes)
        sigLenNum, _ := strconv.ParseUint(sigLenStr, 10, 32)
        sigLen := int(sigLenNum)
	sigBytes := b.Data[53:53 + sigLen]
	sigStr := string(sigBytes)
	sigDecoded, _ := formatting.Decode(formatting.CB58, sigStr)
	dataBytes := b.Data[153:]
	//blockTypeBytes := dataBytes[0]
	/*
	blockLenBytes := dataBytes[1:5]
	blockLenStr := string(blockLenBytes)
	blockLenNum, _ := strconv.ParseUint(blockLenStr, 10, 32)
	blockLen := int(blockLenNum)
	*/
	//messageBytes := dataBytes[:5 + blockLen]
	messageBytes := dataBytes

	if pubkey.Verify(messageBytes, sigDecoded) == false {
		return errInvalidSignature
	}

	// Our block inherits VM from *core.Block.
	// It holds the database we read/write, b.VM.DB
	// We persist this block to that database using VM's SaveBlock method.
	if err := b.VM.SaveBlock(b.VM.DB, b); err != nil {
		return errDatabaseSave
	}

	// Then we flush the database's contents
	return b.VM.DB.Commit()
}
