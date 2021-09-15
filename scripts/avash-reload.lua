print('stopping node 5')
avash_call("procmanager stop node5")
avash_call("procmanager remove node5")


subnet_id = avash_call("varstore print vm subnet_id")
output = avash_call("startnode node5 --db-type=memdb --staking-enabled=true --http-port=9658 --staking-port=9659 --log-level=debug --bootstrap-ips=127.0.0.1:9651 --bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg --staking-tls-cert-file=certs/keys5/staker.crt --staking-tls-key-file=certs/keys5/staker.key --index-enabled --whitelisted-subnets=" .. subnet_id)
