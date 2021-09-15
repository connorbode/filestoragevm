// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package filestoragevm

import (
	"errors"
	"time"
	"strconv"

	"github.com/ava-labs/avalanchego/indexer"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/json"
	"github.com/ava-labs/avalanchego/utils/rpc"
	"github.com/ava-labs/avalanchego/vms/components/core"
	"github.com/ava-labs/avalanchego/vms/platformvm"
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
	errStakingPeriodInvalid = errors.New("staking period must start at least 30 seconds from now and last at least 1 minute")

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

func (b *Block) isStakeBlock() bool {
	return b.getBlockType() == "2"
}

func (b *Block) getStakeNode() string {
	return string(b.Data[158:158+40])
}

func (b *Block) getStakeRewardAddress() string {
	return string(b.Data[158+40 : 158 + 40 + 50])
}

func (b *Block) getStakeStart() int64 {
	startBytes := b.Data[158+40+50:158+40+50+10]
	return b.convertBytesToInt(startBytes)
}

func (b *Block) getStakeEnd() int64 {
	endBytes := b.Data[158+40+50+10:158+40+50+10+10]
	return b.convertBytesToInt(endBytes)
}

func (b *Block) getStakeAmount() int64 {
	amountBytes := b.Data[158+40+50+10+10:158+40+50+10+10+16]
	return b.convertBytesToInt(amountBytes)
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


func wasNodeValidatingAtTime(nodeID string, timestamp int64) bool {
	// get node host? it must be possible
	uri := "http://localhost:9658"
	// get subnet ID, uhoh another thing!!! FK
	subnetID, _ := ids.FromString("2PYeJUhPiTrhe6Yq73okTzayR15U5WUD2R2idfsCLfqahEi5uo")
	timeout := 10 * time.Second
	// 1. instantiate p-chain api
	pChain := platformvm.NewClient(uri, timeout)
	// 2. get height on p-chain api
	height, _ := pChain.GetHeight()
	// 3. instantiate p-chain indexer api

	pChainIndex := indexer.NewClient(uri, "/ext/index/P/block", timeout)
	// 4. start at p-chain top index

	for true {
		res, _ := pChainIndex.GetContainerByIndex(&indexer.GetContainer{
			Index: json.Uint64(height - 1),
			Encoding: formatting.CB58,
		})
		if res.Timestamp.Unix() < int64(timestamp) {
			break
		}
		height -= 1
		if height < 0 {
			// we have a problem, how to raise an error?
		}
	}
	// 5. find the p-chain index that is just before our timestamp
	// 6. use the p-chain api to get the validators active at that timestamp
	// this one isn't implemented in the client
	getValidatorsAtReply := &platformvm.GetValidatorsAtReply{}
	pChainRequester := rpc.NewEndpointRequester(uri, "/ext/P", "platform", timeout)
	pChainRequester.SendRequest("getValidatorsAt", &platformvm.GetValidatorsAtArgs{
		Height: json.Uint64(height),
		SubnetID: subnetID,
	}, getValidatorsAtReply)
	wasValidating := false
	for k := range getValidatorsAtReply.Validators {
		if k == nodeID {
			wasValidating = true
		}
	}

	return wasValidating
}

func (b *Block) getStakeReward() uint64 {
	if time.Now().Unix() <= b.getStakeEnd() {
		return 0
	}

	currTime := b.getStakeStart()
	for currTime < b.getStakeEnd() {
		if !wasNodeValidatingAtTime(b.getStakeNode(), currTime) {
			return 0 // harsh, but 
		}
		currTime += 60 * 5 // verify each 5 minute interval that they were validating
	}

	// I'm not sure whether this actually represents the validator being online,
	// or whether they were just "validating the subnet" at the time.
	// I think we also need some sort of uptime metric in here to verify the
	// node was really online and securing hte network

	// we should also consider the validators stake, b.getStakeAmount(), in this equation
	return uint64(b.getRewardPerSecond() * uint64(b.getStakeEnd() - b.getStakeStart()))
}

func (b *Block) getLockedStake() uint64 {
	currTime := time.Now().Unix()
	if currTime >= b.Timestamp().Unix() && currTime <= b.getStakeEnd() {
		return uint64(b.getStakeAmount())
	}
	return 0
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
	} else if b.isUploadBlock() {
		// upload fees get paid back to the unallocated account
		balance += b.getCostPerUploadBlock()
	} else if b.isStakeBlock() {
		balance -= int64(b.getStakeReward())
	}
	return balance
}

func (b *Block) getUploadSender() string {
	pubkeyBytes := b.Data[0:50]
	return string(pubkeyBytes)
}

func (b *Block) getRewardPerSecond() uint64 {
	// this is just fixed for demo purposes
	return 1
}

func (b *Block) getCostPerUploadBlock() int64 {
	return 1 // this is just fixed for demo purposes
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
		// faucet distributions
		balance += b.getFaucetAmount()
	} else if b.isTransferBlock() {
		// transfers between wallets
		if b.getTransferSender() == account {
			balance -= b.getTransferAmount()
		}
		if b.getTransferRecipient() == account {
			balance += b.getTransferAmount()
		}
	} else if b.isUploadBlock() && b.getUploadSender() == account {
		// actual file uploads
		balance -= b.getCostPerUploadBlock()
	} else if b.isStakeBlock() && b.getStakeRewardAddress() == account {
		// distribution of staking rewards
		balance += int64(b.getStakeReward()) // should be 0 if staking
		balance -= int64(b.getLockedStake()) // should be stake amount if staking, 0 otherwise

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
	messageBytes := dataBytes
	if pubkey.Verify(messageBytes, sigDecoded) == false {
		return errInvalidSignature
	}

	// validate different types of blocks
	if b.isUploadBlock() {
		if parent.getBalance(b.getUploadSender()) < b.getCostPerUploadBlock() {
			return errInsufficientBalance
		}
	} else if b.isFaucetBlock() {
		// faucet, only error is if faucet is empty
		if b.getFaucetAmount() > parent.getUnallocatedBalance() {
			return errFaucetEmpty
		}
	} else if b.isTransferBlock() {
		if b.getTransferAmount() > parent.getBalance(b.getTransferSender()) {
			return errInsufficientBalance
		}
	} else if b.isStakeBlock() {
		if b.getStakeStart() < time.Now().Add(10 * time.Second).Unix() {
			return errStakingPeriodInvalid
		} else if b.getStakeEnd() - b.getStakeStart() < 10 {
			return errStakingPeriodInvalid
		} else if b.getStakeAmount() > parent.getBalance(b.getStakeRewardAddress()) {
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
