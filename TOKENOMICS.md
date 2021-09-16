# Tokenomics ..

So the tokenomics are pretty lacking here and could use some improvement. I wanted to build the architecture out first and didn't have a lot of time to consider tokenomics. But I'll explain the system that was built.

## System Account

All of the funds start in the system account. It's initialized to have a balance of 5000000000000000. My son loves the number 5. And I wanted a big number. That's how I came up with it.

Funds flow out of the system account as follows:

1. Rewards are paid out to stakers
2. Faucet (obviously this would be disabled in a real environment)

In a real deployment, a large portion of the balance would be distributed via airdrop to Validators on the primary subnet, incentivizing them to validate this subnet.

Funds flow back into the system account when users upload files. 

## Uploading Data

There is a fixed cost of 1 token per block to upload data. In a real deployment, the cost would likely have to be much more complex.

## Staking

Validators of the network can stake their funds to earn. Staking happens as follows:

1. Validators must be validating the subnet.
1. Validators submit a staking transaction, which specifies: (1) their NodeID; (2) the address used, which supplies staked funds and also receives rewards; (3) the start time of the staking; (4) the end times of the staking.
1. When the staking period is over, rewards are passed on to the address.

Rewards are only awarded if:

1. The NodeID was indeed validating the Subnet during the staking period. This is done by checking the P-Chain to verify that, for every 5 minute interval, the NodeID was there and validating.
1. The reward period is over.

Current problems with staking:

- Node uptime is not considered at the moment, but it should be factor in the rewards.
- There is no protection against double staking / overlapping staking periods for the same nodes, which would result in 2x rewards. That's an easy fix that should be implemented.
- There is no authentication for NodeIDs. So, for example, if I know the NodeID of your node, I can use my own rewards account and as long as you're staking on the subnet I can claim your rewards. That's a huge issue, but I wasn't able to find the mechanism for authenticating the NodeID. My assumption is that the NodeID is a prefixed public key, and there is a corresponding private key somewhere which can be used to sign the staking message, authenticating the node & mitigating this issue.

So all told, that's the system. I think the incentives are there to bring validators to the network but they would just need to be balanced so that the economics work long-term.





