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
	errFaucetEmpty       = errors.New("faucet is out of funds sorry bud")
	errInsufficientBalance = errors.New("insufficient balance for transfer")

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

func (b *Block) getBlockType() string {
	blockTypeBytes := b.Data[153]
	return string(blockTypeBytes)
}

func (b *Block) isUploadBlock() bool {
	return b.getBlockType() == "0"
}

func (b *Block) isFaucetBlock() bool {
	return b.getBlockType() == "9"
}

func (b *Block) isTransferBlock() bool {
	return b.getBlockType() == "1"
}

func (b *Block) getTransferAmount() int64 {
	amountBytes := b.Data[158:158 + 16]
	return b.convertBytesToInt(amountBytes)
}

func (b *Block) getTransferSender() string {
	return string(b.Data[174:174 + 50])
}

func (b *Block) getTransferRecipient() string {
	return string(b.Data[174 + 50:174 + 50 + 50])
}

func (b *Block) getFaucetAmount() int64 {
	amountBytes := b.Data[158:158 + 16]
	return b.convertBytesToInt(amountBytes)
}

func (b *Block) getFaucetRecipient() string {
	return string(b.Data[174:174 + 50])
}

// convenience method to take a block of bytes
// and pull an integer out of them
func (b *Block) convertBytesToInt(bytes []byte) int64 {
	str := string(bytes)
	num, _ := strconv.ParseUint(str, 10, 64)
        return int64(num)
}


// returns the unallocated balance from the original funds on the blockchain
// these get allocated via faucet or by validators earning rewards
func (b *Block) getUnallocatedBalance() int64 {
	var balance int64
	if b.Parent().String() == "11111111111111111111111111111111LpoYY" {
		balance = 5000000000000000
	} else {
		parentBlock, _ := b.VM.GetBlock(b.Parent())
		parent, _ := parentBlock.(*Block)
		balance = parent.getUnallocatedBalance()
	}
	if b.isFaucetBlock() {
		balance -= b.getFaucetAmount()
	}
	return balance
}

func (b *Block) getBalance(account string) int64 {
	var balance int64
	if b.Parent().String() == "11111111111111111111111111111111LpoYY" {
		balance = 0
	} else {
		parentBlock, _ := b.VM.GetBlock(b.Parent())
		parent, _ := parentBlock.(*Block)
		balance = parent.getBalance(account)
	}
	if b.isFaucetBlock() && b.getFaucetRecipient() == account {
		balance += b.getFaucetAmount()
	} else if b.isTransferBlock() {
		if b.getTransferSender() == account {
			balance -= b.getTransferAmount()
		}
		if b.getTransferRecipient() == account {
			balance += b.getTransferAmount()
		}
	}
	return balance
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

	// validate different types of blocks
	if b.isUploadBlock() {
		// upload fauc
	} else if b.isFaucetBlock() {
		// faucet, only error is if faucet is empty
		if b.getFaucetAmount() > b.getUnallocatedBalance() {
			return errFaucetEmpty
		}
	} else if b.isTransferBlock() {
		if b.getTransferAmount() > b.getBalance(b.getTransferSender()) {
			return errInsufficientBalance
		}
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
