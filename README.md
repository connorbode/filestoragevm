# FilestorageVM

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
