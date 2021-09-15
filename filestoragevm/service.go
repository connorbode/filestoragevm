// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package filestoragevm

import (
	"errors"
	"fmt"
	"strconv"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/json"
)

var (
	errBadData     = errors.New("data must be base 58 repr. of 32 bytes")
	errNoSuchBlock = errors.New("couldn't get block from database. Does it exist?")
)

// Service is the API service for this VM
type Service struct{ vm *VM }

// ProposeBlockArgs are the arguments to function ProposeValue
type ProposeBlockArgs struct {
	// Data in the block. Must be base 58 encoding of 32 bytes.
	Data string `json:"data"`
}

// ProposeBlockReply is the reply from function ProposeBlock
type ProposeBlockReply struct{ Success bool }

// ProposeBlock is an API method to propose a new block whose data is [args].Data.
// [args].Data must be a string repr. of a 32 byte array
func (s *Service) ProposeBlock(_ *http.Request, args *ProposeBlockArgs, reply *ProposeBlockReply) error {
	bytes, err := formatting.Decode(formatting.CB58, args.Data)
	if err != nil || len(bytes) != dataLen {
		return errBadData
	}
	var data [dataLen]byte         // The data as an array of bytes
	copy(data[:], bytes[:dataLen]) // Copy the bytes in dataSlice to data
	s.vm.proposeBlock(data)
	reply.Success = true
	return nil
}

// APIBlock is the API representation of a block
type APIBlock struct {
	Timestamp json.Uint64 `json:"timestamp"` // Timestamp of most recent block
	Data      string      `json:"data"`      // Data in the most recent block. Base 58 repr. of 5 bytes.
	ID        string      `json:"id"`        // String repr. of ID of the most recent block
	ParentID  string      `json:"parentID"`  // String repr. of ID of the most recent block's parent
}

// GetBlockArgs are the arguments to GetBlock
type GetBlockArgs struct {
	// ID of the block we're getting.
	// If left blank, gets the latest block
	ID string
}

// GetBlockReply is the reply from GetBlock
type GetBlockReply struct {
	APIBlock
}

// GetBlock gets the block whose ID is [args.ID]
// If [args.ID] is empty, get the latest block
func (s *Service) GetBlock(_ *http.Request, args *GetBlockArgs, reply *GetBlockReply) error {
	// If an ID is given, parse its string representation to an ids.ID
	// If no ID is given, ID becomes the ID of last accepted block
	var id ids.ID
	var err error
	if args.ID == "" {
		id, err = s.vm.LastAccepted()
		if err != nil {
			return fmt.Errorf("problem finding the last accepted ID: %s", err)
		}
	} else {
		id, err = ids.FromString(args.ID)
		if err != nil {
			return errors.New("problem parsing ID")
		}
	}

	// Get the block from the database
	blockInterface, err := s.vm.GetBlock(id)
	if err != nil {
		return errNoSuchBlock
	}

	block, ok := blockInterface.(*Block)
	if !ok { // Should never happen but better to check than to panic
		return errBadData
	}

	// Fill out the response with the block's data
	reply.APIBlock.ID = block.ID().String()
	reply.APIBlock.Timestamp = json.Uint64(block.Timestamp().Unix())
	reply.APIBlock.ParentID = block.Parent().String()
	reply.Data, err = formatting.EncodeWithChecksum(formatting.CB58, block.Data[:])

	return err
}

type CreateAddressArgs struct {
}

type CreateAddressReply struct {
	PrivateKey string `json:"privateKey"`
	PublicKey string `json:"publicKey"`
}

func (s *Service) CreateAddress(_ *http.Request, args *CreateAddressArgs, reply *CreateAddressReply) error {
	var err error
	factory := crypto.FactorySECP256K1R{}
	skIntf, err := factory.NewPrivateKey()
	if err != nil {
		return fmt.Errorf("problem generating private key: %w", err)
	}
	sk := skIntf.(*crypto.PrivateKeySECP256K1R)
	privKeyStr, _ := formatting.EncodeWithChecksum(formatting.CB58, sk.Bytes())
	pubKeyStr, _ := formatting.EncodeWithChecksum(formatting.CB58, sk.PublicKey().Bytes())
	reply.PublicKey = pubKeyStr
	reply.PrivateKey = privKeyStr
	return err
}

type GetBlockHeightArgs struct {
}

type GetBlockHeightReply struct {
	BlockHeight string `json:"blockHeight"`
}

func (s *Service) GetBlockHeight(_ *http.Request, args *GetBlockHeightArgs, reply *GetBlockHeightReply) error {
	var err error
	var id ids.ID
	id, err = s.vm.State.GetLastAccepted(s.vm.DB)
	reply.BlockHeight = id.String()
	return err
}

type GetStorageCostArgs struct {
}

type GetStorageCostReply struct {
	Cost string `json:"cost"`
}

func (s *Service) GetStorageCost(_ *http.Request, args *GetStorageCostArgs, reply *GetStorageCostReply) error {
	var err error
	reply.Cost = "0" // TODO
	return err
}

type GetUnallocatedFundsArgs struct {
}

type GetUnallocatedFundsReply struct {
	UnallocatedFunds int64 `json:"unallocatedFunds"`
}

func (s *Service) GetUnallocatedFunds(_ *http.Request, args *GetUnallocatedFundsArgs, reply *GetUnallocatedFundsReply) error {
	var err error
	id, _ := s.vm.State.GetLastAccepted(s.vm.DB)
	coreBlock, _ := s.vm.GetBlock(id)
	block, _ := coreBlock.(*Block)
	reply.UnallocatedFunds = block.getUnallocatedBalance()
	return err
}

type GetBalanceArgs struct {
	Account string
}

type GetBalanceReply struct {
	Balance int64 `json:"balance"`
}

func (s *Service) GetBalance(_ *http.Request, args *GetBalanceArgs, reply *GetBalanceReply) error {
	var err error
	id, _ := s.vm.State.GetLastAccepted(s.vm.DB)
	coreBlock, _ := s.vm.GetBlock(id)
	block, _ := coreBlock.(*Block)
	reply.Balance = block.getBalance(args.Account)
	return err
}

type DebugPayloadArgs struct {
	Payload string
}

type DebugPayloadReply struct {
	SigValid bool `json:"sigValid"`
	Sig string `json:"sig"`
	Message string `json:"message"`
	MessageLength int `json:"messageLength"`
	Pubkey string `json:"pubkey"`
}

func (s *Service) DebugPayload(_ *http.Request, args *DebugPayloadArgs, reply *DebugPayloadReply) error {
	var err error
	data, err := formatting.Decode(formatting.CB58, args.Payload)
	factory := crypto.FactorySECP256K1R{}
	pubkeyBytes := data[0:50]
	pubkeyDecoded, _ := formatting.Decode(formatting.CB58, string(pubkeyBytes))
	pubkey, _ := factory.ToPublicKey(pubkeyDecoded)
	pubkeyStr, _ := formatting.EncodeWithChecksum(formatting.CB58, pubkey.Bytes())
	reply.Pubkey = pubkeyStr
	sigLenBytes := data[50:53]
	sigLenStr := string(sigLenBytes)
        sigLenNum, _ := strconv.ParseUint(sigLenStr, 10, 32)
        sigLen := int(sigLenNum)
	sigBytes := data[53:53 + sigLen]
	sigStr := string(sigBytes)
	sigDecoded, _ := formatting.Decode(formatting.CB58, sigStr)
	reply.Sig = sigStr
	dataBytes := data[153:]
	//blockTypeBytes := dataBytes[0]
	/*
	blockLenBytes := dataBytes[1:5]
	blockLenStr := string(blockLenBytes)
	blockLenNum, _ := strconv.ParseUint(blockLenStr, 10, 32)
	blockLen := int(blockLenNum)
	*/
	//messageBytes := dataBytes[:5 + blockLen]
	messageBytes := dataBytes
	reply.MessageLength = len(messageBytes)
	messageStr := string(messageBytes)
	reply.Message = messageStr
	reply.SigValid = pubkey.Verify(messageBytes, sigDecoded)
	return err
}


