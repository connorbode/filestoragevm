# CLI!

So.. this CLI is just a Python script that launches a debugger. Then you have the `api` object instantiated and you can communicate with the chain. Before I get into the methods, let's talk about the weird stuff.

### Crypto / signing

There's a Python library [here](https://github.com/ludbb/secp256k1-py) which claims to handle the secp256k1 stuff but I couldn't get it to work. I learned about these things in phases, so for the first phase where I was just generating accounts, I did it as an RPC call which returns a public key and a private key to the CLI. That really doesn't make a lot of sense, it should be generated locally. However, I left it there as I was running out of time. Next, when I had to authenticate messages, I knew it was something I would have to do locally. When Python didn't work, I decided using a golang program would be the simplest thing. 

So, we have a file called `keys.go`. That file handles generating signatures for the messages. It should also handle key generation, but that's something that I didn't port over to here. `cli.py` makes calls to `keys.go` and passes sensitive info via a temporary file. Not super secure, but it works for now.


## Usage

To open the CLI, run `python3 cli.py <blockchain_id>`. This will open a connection on avash.

## Instantiating

So if you just launch the CLI, it's already hooked up to avash. But you can also call `FilestorageAPI(host, blockchain_id, block_timeout)` to create a connection to any node. Once I finish syncing Fuji I'll try to update this with instructions to connect.

- `blockchain_id` is the ID of the custom blockchain that was created on the network.
- `block_timeout` is the number of seconds to wait before assuming that a block failed to Verify
- `host` points to the avalanchego node.

## Methods

### `api.create_account()`

Returns a keypair (public key, private key) which represent your account on the chain. Optionally call `api.create_account(True)` to automatically use these credentials for the CLI.

The private key is not used for any of the methods in the CLI (though it is used to authenticate messages in the background). The public key is used anywhere where you see the `account` variable. In these instances, you can pass `keypair[0]`.

### `api.set_credentials(keypair)`

Sets the credentials that are used to transact with the server.

### `api.get_balance(public_key)`

This returns the balance of a given account.

For example:

```
keypair = api.create_account()
account = keypair[0] // public key
balance = api.get_balance(account)
```

### `api.get_storage_cost()`

Returns the token cost of storing one block of data. At the moment this returns a fixed price of `1`, however the tokenomics could certainly be improved.

Data transactions will check account balances are sufficient for uploading the entire amount of data before proceeding.

### `api.upload_data(data_string)`

Uploads a string of data to the blockchain in one or more blocks of data. Verifies your account has enough tokens to do so before proceeding.

Returns a list of block IDs that can be used to fetch the data.

e.g.

```
block_ids = api.upload_data('Hello, this is just some text.')
```

### `api.download_data(block_ids)`

Downloads data that's been uploaded to the chain, using `block_ids` that were returns from the `api.upload_data` method.

e.g.

```
original_str = 'Hi, this is a test message'
block_ids = api.upload_data(my_str)
downloaded_str = api.download_data(block_ids)
assert downloaded_str == original_str  ## >> True
```

### `api.upload_file('./path/to/file.png')`

Uploads a file to the chain (base64 encodes it first). This is a convenience method that wraps around `api.upload_data`

Returns a list of block IDs that can be used to fetch the data.


### `api.download_file(block_ids, './path/to/output/file.png')`

Downloads a file from the chain and saves it to disk. This is a convenience method wrapped around `api.download_data`

e.g.

```
local_file = './original.png'
block_ids = api.upload_file(local_file)
api.download_file(block_ids, './duplicate.png')
# Check the images, they will be the same.
```


### `api.faucet(amount, recipient)`

This method transfers funds from the "system account" (which carries all tokens to start) into the `recipient` account. 

e.g.

```
keypair = api.create_account()
account = keypair[0]
api.faucet('2000', account)
```

### `api.get_unallocated_balance()`

Returns the balance of the "system account". See [TOKENOMICS](https://github.com/connorbode/filestoragevm/blob/main/TOKENOMICS.md) for more details.


### `api.transfer(amount, recipient)`

Transfers `amount` from your account to `recipient`.

e.g.

```
keypair = api.create_account()
account = keypair[0]
api.faucet('2000', account)
api.set_credentials(keypair)
other = api.create_account()
other_account = other[0]
api.transfer('2000', other_account)
api.get_balance(account)
> 0, you transferred it all
api.get_balance(other_account)
> 2000, you transferred it all
```

### `api.stake(node_id, amount, start, end)`

Validators of the subnet need to stake to be able to earn tokens for validating the subnet.

At the moment, all validators just earn an equal amount per second for staking. 

There are a couple of problems with this at the moment:

1) There's quite likely a bug where validators can overlap (so you can probably submit two staking transactions to earn double. This can be resolved.

2) It is probably exploitable as there is nothing authenticating the node. So, if I know another person's node is validating the subnet, I can "stake" for their node and earn tokens. This is because the `stake` method identifies the account that receives the rewards. In order to resolve this, I would need a way of proving I am authorized to stake on behalf of a given node. I think that there are likely signing keys somewhere, but I wasn't sure how to access them.



