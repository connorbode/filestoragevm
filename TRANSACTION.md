# Transaction Protocol

## Data

All of the data is stored in blocks of a fixed size. At the moment, the block size is 4096 bytes. This was chosen to be a balance between (1) being able to upload a reasonable amount of data in one block; and (2) not bloating the chain with too many sparse blocks for transfer / staking transactions.


I'll try my best to explain how things are packaged, but I'm doing so just by reading the code so I may be incorrect in some places.

Let's go layer by layer, starting from the top.

### Layer 1: CB58 Encoding

At the top level, we have the whole payload CB58 encoded. This makes it easy to transfer any type of data over the network, but honestly I mostly just chose it because that's what the TimestampVM used.

### Layer 2: Authentication

This layer authenticate's the user's account. I still don't have any authentication for NodeIDs, and that's a huge issue, but at least we're authenticating user accounts for balance modifying transactions.

- Bytes 0:50 are the public key (CB58 encoded)
- Bytes 50:53 are an integer representing the length of the signature bytes
- Bytes 53:153 are the signature of the message (CB58 encoded)
- Bytes 153:<end> are the message, which is described below.

### Layer 3: Message

Message starts at offset 153, but we'll consider only the slice starting at offset 153. This layer describes the type of message and the content length (to tell us how many null bytes we have padding the end which are not considered part of the payload).

- Byte 0 represents the message type. There are four types of messages: __0: Data Chunk__, __1: Balance Transfer__, __2: Stake__, __9: Faucet__.
- Bytes 1:5 are an integer representing the content length
- Bytes 5:<5 + content length> are the actual message data, described again below.
- Bytes <5 + content length>:<end> is padded with \x00

### Layer 4: Specific Message Type

This is dependent on the message type from layer 3.

#### Type 0: Data Chunk

- Bytes 0:16 are a file ID. The idea was that this would uniquely represent the file, so that we could piece together files just having the blockchain. Combined with the public key in the authentication layer, we can piece together files as long as the user properly generates unique file IDs. Uniqueness is not enforced, and this is not the primary mechanism used to retrieve files.
- Bytes 16:24 are an integer representing the chunk number, so blocks could technically be uploaded out of order and could still be retrieved.
- Bytes 24:<end> are the actual data

#### Type 1: Balance Transfer

- Bytes 0:16 are an integer representing the amount to transfer
- Bytes 16:66 represent the sender of the transfer. This is actually likely redundant because the sender would have to be the signer. And I've just realized that there is no protection against sending someone else's funds!
- Bytes 66:116 represent the recipient of the transfer.

#### Type 2: Stake

- Bytes 0:40 are the Node ID that is staking, which must be validating this subnet in order to receive rewards.
- Bytes 40:90 are the account address which will stake funds + will receive rewards
- Bytes 90:100 are an integer representing the time when staking starts
- Bytes 100:110 are an integer representing the time when staking ends
- Bytes 110:126 are an integer representing the amount of funds that will be staked (which must be less than the balance of the account)

#### Type 9: Faucet

- Bytes 0:16 are an integer representing the amount of funds to be transfered
- Bytes 16:66 are the address of the account receiving the funds

## Security Issues

- There is no protection against replay attacks. Perhaps this can be remedied by adding a time-based nonce? But I didn't have time to explore.
- I can probably spend anyone's balance, because I realized in writing this document that the sender of a transaction does not need to match the signer of the message.
- Keypair is currently generated on the server. This could be done on the client side, but after I figured out how I ran out of time to implement.

## Scalability Issues

- There is no caching implemented. Everything is computed by just traversing the chain entirely. This isn't scalable at all.
- The chain grows infinitely and there are no attempts to prune data
- There are lots of sparse blocks

## Crypto stuff

All of the crypto stuff (in the VM and also for the CLI, locally) is done using the libraries from avalanchego repository, specifically `crypto.FactorySECP256K1R`.
