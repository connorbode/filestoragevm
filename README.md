# FilestorageVM

This project was implemented as part of a hackathon. The goal was to create a custom virtual machine / subnet which run on Avalanche. I created a chain which allows users to store arbitrary data (e.g. for files or other things). Validators of the subnet are able to earn rewards for validating.

For further reading:

- ["Tokenomics" - lol](https://github.com/connorbode/filestoragevm/blob/main/TOKENOMICS.md)
- ["Data storage / transaction details](https://github.com/connorbode/filestoragevm/blob/main/TRANSACTION.md)

This project currently is only local. I intend to deploy it to Fuji, but of course I forgot to sync the chain and now it's taking forever. Will update with more details if I put it live. 


## Validating on Fuji

1. You'll need to copy the binary out of `/bin` (or compile yourself using the build script) into your `avalanchego/plugins` directory.
1. Add your node as a validator on the primary subnet
1. Add your node as a validator on the subnet wtyFZF1UUyjyXeiVCP5Vd5dpQ6KPCmG24qW3agbSRdtVEh7Jb

Then, to validate on Fuji, we need to run avalanchego as follows:

`avalanchego --network-id=fuji --whitelisted-subnets=wtyFZF1UUyjyXeiVCP5Vd5dpQ6KPCmG24qW3agbSRdtVEh7Jb --index-enabled`

Unfortunately `--index-enabled` requires resyncing the whole chain.


## Connecting to Fuji

Will provide more info on this once I resync the chain with index-enabled haha.

```
SUBNET_ID=wtyFZF1UUyjyXeiVCP5Vd5dpQ6KPCmG24qW3agbSRdtVEh7Jb
BLOCKCHAIN_ID=6XHJC4cJVzSyF4iaA5TugsnWzn3LTY1Y5DwGwaTYYUrWuwQJ7
```

## Installation

1. Clone repo
1. Install avash & avalanchego (v1.5.3). I have them in /avash & /avalanchego-v1.5.3 (relative to the root of this project). It might work if you put them elsewhere but no guarantees.
1. Install the required tools (see Required Tools section below)
1. Configure ~/.avash.yaml (see Avash Configuration section below)
1. Set up environment variables (see Environment section below)
1. Install the Python requirements: `pip install -r cli/requirements.txt`


### Operating System Note

This won't work on windows probably, maybe on other OS. Mostly because of path separators but maybe there are other issues.


### Required Tools

Ok so, this project is all over the place. Good luck, you'll need:

- Golang
- Python & pip (I'm running 3.7.x)
- NodeJS (I'm running v15.0.1, this isn't *really* a hard requirement, but it's used in one of the scripts)


### Avash Configuration

Edit your `~/.avash.yaml` file. Make sure that you set `avalancheLocation: <absolute_path_to_avalanche_go>`.


### Environment

The following environment variables need to be set

- `AVALANCHEGO_DIR`: absolute path to Avalanche Go
- `FILESTORAGEVM_DIR`: absolute path to the root directory of this project


## Running Locally

### Launch Avash & the local dev network

1. From the avash directory, ./avash
1. In the avash CLI, type `runscript ../scripts/avash-start.lua`. This will only work if you installed avash into `/avash` relative to the project root. Otherwise, replace the `..` with the full path to the project root.
1. Wait for everything to boot. At the end you'll see `done, we are validating!` along with the blockchain id and the subnet id.
1. Leave this open
1. Copy your `blockchain_id` - you'll need it to run the CLI

### Building the VM

1. From the project root, run `bash build.sh`
1. From the Avash shell, type `runscript ../scripts/avash-reload.lua` (this reboots the node validating the subnet)

### Running the CLI

From the `/cli` directory, run `python3 cli.py <blockchain_id>`

[More details on the CLI are available here](https://github.com/connorbode/filestoragevm/blob/main/cli/README.md)


