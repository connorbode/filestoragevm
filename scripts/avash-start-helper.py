import requests
import json
import os
import pathlib
import sys
from time import sleep, time

#  __          __     _____  _   _ _____ _   _  _____      _       _
#  \ \        / /\   |  __ \| \ | |_   _| \ | |/ ____|  /\| |/\ /\| |/\
#   \ \  /\  / /  \  | |__) |  \| | | | |  \| | |  __   \ ` ' / \ ` ' /
#    \ \/  \/ / /\ \ |  _  /| . ` | | | | . ` | | |_ | |_     _|_     _|
#     \  /\  / ____ \| | \ \| |\  |_| |_| |\  | |__| |  / , . \ / , . \
#      \/  \/_/    \_\_|  \_\_| \_|_____|_| \_|\_____|  \/|_|\/ \/|_|\/
# 
#
#
#          THIS FILE IS NOT MEANT TO BE RUN DIRECTLY. 
# 			RUN avash-start.lua FROM THE AVASH CONSOLE.
#
#

AVALANCHEGO = os.environ.get('AVALANCHEGO_DIR') + '/avalanchego'
SCRIPTS_DIR = pathlib.Path(__file__).parent.resolve()
USERNAME = 'username'
PASSWORD = 'unaware-module-enunciate'
WHALE_PRIVATE_KEY = 'PrivateKey-ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN'
VM_ID = 'qAyzuhzkcQQsAYQP3iibkD28DqXTS8cRsFC8PR3LuqebWVS2Q'
TMPFILE = f'{SCRIPTS_DIR}/tmp'

class RPC:
	def __init__(self, url):
		self.url = url

	def send(self, method, params, authenticate=None):
		base = self.url
		paths = {
			'avm': '/ext/bc/X',
			'health': '/ext/health',
			'platform': '/ext/platform',
			'keystore': '/ext/keystore',
			'info': '/ext/info',
			'timestampvm': '/ext/vm/tGas3T58KzdjLHhBDMnH2TvrddhqTji5iZAMZ3RXs2NLpSnhH',
			'filestoragevm': f'/ext/vm/{VM_ID}',
		}

		# add credentials to the payload by default
		if authenticate == None:
			params['username'] = USERNAME
			params['password'] = PASSWORD
		
		payload = {
			'jsonrpc': '2.0',
			'method': method,
			'params': params,
			'id': 1,
		}
		
		path = paths[method.split('.')[0]]
		url = f'{base}{path}'
		res = requests.post(url, json=payload)
		res.raise_for_status()
		return res.json()

	def check_tx(self, tx_id):
		print(self.send('platform.getTxStatus', {
			'txID': tx_id 
		}))

	def wait_for_tx_commit(self, tx_id):
		while True:
			tx = self.send('platform.getTxStatus', {
				'txID': tx_id
			})
			if tx['result']['status'] == 'Committed':
				break
			elif tx['result']['status'] == 'Dropped':
				raise Exception('Dropped')
			sleep(1)

	def wait_for_bootstrap(self):
		bootstrapped = False
		print('waiting for node to bootstrap')
		while True:
			try:
				out = self.send('health.getLiveness', {})
				if out['result']['healthy']:
					bootstrapped = True
			except:
				pass
			if bootstrapped:
				break
			sleep(1)
	
	def import_whale(self):
		self.send('keystore.createUser', {
			'username': USERNAME,
			'password': PASSWORD,
		})
		return self.send('platform.importKey', {
			'privateKey': WHALE_PRIVATE_KEY
		})['result']['address']
	
	def get_node_id(self):
		return self.send('info.getNodeID', {})['result']['nodeID']

def wait_for_bootstrap():
	ports = [
		'9650',
		'9652', 
		'9654',
		'9656',
curl -X POST --data '{
    "jsonrpc": "2.0",
    "method": "platform.createAddress",
    "params": {
        "username":"myUsername",
        "password":"myPassword"
    },
    "id": 1
}' -H 'content-type:application/json;' 127.0.0.1:9650/ext/bc/P
		'9658',
	]
	while True:
		healthy_count = 0
		total_count = len(ports)
		for port in ports:
			out = RPC(f'http://127.0.0.1:{port}').send('health.getLiveness', {})
			if out['result']['healthy']:
				healthy_count += 1
		if healthy_count == total_count:
			break
		print(f'{healthy_count}/{total_count} healthy nodes')
		sleep(10)

def step_one():
	sleep(5)
	wait_for_bootstrap()
	rpc = RPC('http://127.0.0.1:9658')

	print('creating accounts')

	# create our user
	rpc.send('keystore.createUser', {
		'username': USERNAME,
		'password': PASSWORD,
	})

	# import the whale account to the p-chain
	whale = rpc.import_whale()

	print('creating subnet')

	# get initial subnet IDs
	before_subnet_ids = set([ss['id'] for ss in rpc.send('platform.getSubnets', {})['result']['subnets']])
	create_subnet = rpc.send('platform.createSubnet', {})

	# poll subnets until the new one is created
	while True:
		after_subnet_ids = set([ss['id'] for ss in rpc.send('platform.getSubnets', {})['result']['subnets']])
		diff = after_subnet_ids.difference(before_subnet_ids)
		if len(diff) > 0:
			break
		sleep(1)
	subnet_id = list(diff)[0]

	with open(TMPFILE, 'w') as f:
		f.write(subnet_id)
	
	print('adding subnet validator')
	start_time = int(time()) + 60
	node_id = rpc.get_node_id()
	payload = {
		'nodeID': node_id,
		'subnetID': subnet_id,
		'startTime': start_time, # 30 seconds from now
		'endTime': int(time()) + 60 * 60 * 24 * 30, # 30 days from now
		'weight': 100, # ???
		'changeAddr': whale,
	}
	add_validator = rpc.send('platform.addSubnetValidator', payload)
	rpc.wait_for_tx_commit(add_validator['result']['txID'])

def step_two():
	if os.path.exists(TMPFILE): os.remove(TMPFILE)
	sleep(5)
	wait_for_bootstrap()
	""" submit the node for validation """
	rpc = RPC('http://localhost:9658')
	whale = rpc.import_whale()
	node_id = rpc.get_node_id() 
	subnet_id = sys.argv[2]

	output = rpc.send('platform.getCurrentValidators', {
		'subnetID': subnet_id,
	})

	print('creating the blockchain')
	output = rpc.send('platform.createBlockchain', {
		'subnetID': subnet_id,
		'vmID': VM_ID,
		'name': 'Test name',
		'genesisData': 'fP1vxkpyLWnH9dD6BQA',
	})
	rpc.wait_for_tx_commit(output['result']['txID'])

	print('verifying blockchain is deployed')
	blockchains = rpc.send('platform.getBlockchains', {})
	found = False
	blockchain_id = None
	for chain in blockchains['result']['blockchains']:
		if chain['vmID'] == VM_ID:
			found = True
			blockchain_id = chain['id']
	if not found:
		raise Exception('Blockchain not found..')
	
	print('verifying blockchain is being validated')
	while True:
		status = rpc.send('platform.getBlockchainStatus', {
			'blockchainID': blockchain_id
		})
		if status['result']['status'] == 'Validating':
			break
		elif status['result']['status'] != 'Syncing':
			print(status)
		sleep(1)
	print('done, we are validating!')
	print('blockchain id: ', blockchain_id)
	print('subnet id: ', subnet_id)

steps = {
	'1': step_one,
	'2': step_two,
}

print('running step ' + sys.argv[1])
steps[sys.argv[1]]()
