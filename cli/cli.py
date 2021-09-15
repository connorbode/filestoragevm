import requests
import sys
import math
import time
import secrets
import base64
import cb58ref
import hashlib
import json
import os
from pprint import pprint


class API:
	def __init__(self, host, method_prefix, vm_id, blockchain_id):
		self.host = host
		self.method_prefix = method_prefix
		self.vm_id = vm_id
		self.blockchain_id = blockchain_id
	
	def _call(self, path, method_prefix, method, params):
		url = f'{self.host}{path}'
		data = {
			'jsonrpc': '2.0',
			'method': f'{method_prefix}.{method}',
			'params': params,
			'id': 1
		}
		res = requests.post(url, json=data)
		res.raise_for_status()
		try:
			return res.json()
		except:
			print('failed to decode response')
			print(path, method, params)
			print(res.status_code)
			print(res.text)
	
	def _call_vm(self, method, params):
		path = f'/ext/vm/{self.vm_id}'
		return self._call(path, self.method_prefix, method, params)
	
	def _call_bc(self, method, params):
		path = f'/ext/bc/{self.blockchain_id}'
		return self._call(path, self.method_prefix, method, params)
	
	def _call_p_index(self, method, params):
		path = f'/ext/index/P/block'
		return self._call(path, 'index', method, params)
	
	def _call_p(self, method, params):
		path = f'/ext/P'
		return self._call(path, 'platform', method, params)
	
	def encode(self, data, length=None):
		payload = {
			'data': data
		}
		if length is not None:
			payload['length'] = length
		out = self._call_vm('encode', payload)
		return out['result']['bytes']
	
	def decode(self, data):
		out = self._call_vm('decode', {
			'bytes': data
		})
		return out['result']['data']
	
	def p_index_get_last_accepted(self):
		return self._call_p_index('getLastAccepted', {
			'encoding': 'cb58',
		})
	
	def p_index_get_container_by_index(self, index):
		return self._call_p_index('getContainerByIndex', {
			'index': index,
			'encoding': 'cb58'
		})
	
	def p_get_validators_at(self, height, subnet_id=None):
		payload = {
			'height': height
		}
		if subnet_id is not None: payload['subnetID'] = subnet_id
		return self._call_p('getValidatorsAt', payload)
	
	def p_get_height(self):
		return self._call_p('getHeight', {})
	

class FilestorageAPI(API):
	BLOCK_SIZE = 4096
	DATA_ALLOWANCE_PER_BLOCK = 3914

	def __init__(self, host, bc_id, block_timeout=None):
		if block_timeout is None: block_timeout = 5
		self.block_timeout = block_timeout
		vm_id = 'qAyzuhzkcQQsAYQP3iibkD28DqXTS8cRsFC8PR3LuqebWVS2Q'
		method_prefix = 'filestoragevm'
		super().__init__(host, method_prefix, vm_id, bc_id)
	
	def propose_block(self, data):
		result = self._call_bc('proposeBlock', {
			'data': data
		})['result']
		return result
	
	def set_credentials(self, keypair):
		# (public_key, private_key)
		self.keypair = keypair
	
	def get_latest_block_id(self):
		out = self._call_bc('getBlockHeight', {})
		return out['result']['blockHeight']
	
	def get_block(self, block_id):
		out = self._call_bc('getBlock', {
			'id': block_id
		})
		return out['result']
	
	def get_block_id_from_data(self, data, after_block_id):
		timeout = self.block_timeout
		iterations = 0
		while True:
			block_id = self.get_latest_block_id()
			while block_id != after_block_id:
				block = self.get_block(block_id)
				if block['data'] == data:
					return block_id
				block_id = block['parentID']
			time.sleep(1)
			iterations += 1
			if iterations > timeout:
				raise Exception('Timeout, block probably was not accepted')

	def get_balance(self, account=None):
		if account is None: account = self.keypair[0]
		out = self._call_bc('getBalance', {
			'account': account
		})
		return out['result']['balance']
	
	def get_storage_cost(self):
		""" returns the price to store one upload block """
		out = self._call_bc('getStorageCost', {})
		return out['result']['cost']
	
	def pack_block(self, block_type, block_data):
		block_types = [
			0, # data chunk
			1, # balance transfer
			2, # stake
			9, # faucet
		]
		if block_type not in block_types:
			raise Exception('no, bad coder, do it right.')
		data_len = str(len(block_data))
		while len(data_len) < 4:
			data_len = '0' + data_len
		data = str(block_type) + data_len + block_data
		while len(data) < FilestorageAPI.BLOCK_SIZE - 153:
			data += '\x00'
		signed_data = self.sign(data)
		payload = self.encode(signed_data, FilestorageAPI.BLOCK_SIZE)
		return payload
	
	def unpack_headers(self, data, sizes):
		previous = 0
		sections = []
		for size in sizes:
			section = data[previous:previous+size]
			sections.append(section)
			previous += size
		return sections
	
	def unpack_data_block(self, data):
		file_id = data[0:16]
		chunk_number = int(data[16:24])
		chunk = data[24:]
		output = [file_id, chunk_number, chunk]
		return output
	
	def unpack_block(self, block):
		data = self.decode(block)
		pubkey = data[:50]
		sig_len = data[50:53]
		sig = data[53:153]
		data = data[153:]
		block_type = data[0]
		block_length = int(data[1:5])
		block_data = data[5:5 + block_length]

		output = [block_type]
		if block_type == '0':
			output += self.unpack_data_block(block_data)
		elif block_type == '9':
			output += self.unpack_faucet_block(block_data)
		else:
			output += [block_data]
		return output
	
	def upload_block(self, payload):
		latest_block_id = self.get_latest_block_id()
		result = self.propose_block(payload)
		return self.get_block_id_from_data(payload, latest_block_id)
	
	def upload_data_chunk(self, file_id, chunk_number, chunk):
		chk_num = str(chunk_number)
		while len(chk_num) < 8:
			chk_num = '0' + chk_num
		input_data = [file_id, chk_num, chunk]
		data = ''.join(input_data)
		payload = self.pack_block(0, data)
		return self.upload_block(payload)
	
	def upload_data(self, data, force=None):
		file_id = secrets.token_hex(8)
		number_of_chunks = math.ceil(len(data) / FilestorageAPI.DATA_ALLOWANCE_PER_BLOCK)
		upload_cost = self.get_storage_cost()
		balance = self.get_balance()
		if number_of_chunks * upload_cost > balance:
			if force is None:
				print('Balance not enough to upload data. Quitting. Pass force=True to bypass.')
				raise Exception('BALANCE_NOT_ENOUGH')
			else:
				print('Balance is not enough to upload data, but we are bypassing because you passed force=True')
		chunk_num = 0
		offset_size = FilestorageAPI.DATA_ALLOWANCE_PER_BLOCK
		block_ids = []
		chunks = []
		uploaded_chunks = []
		while True:
			print(f'uploading chunk {chunk_num + 1}/{number_of_chunks}')
			chunk = data[chunk_num * offset_size : (chunk_num + 1) * offset_size]
			chunks.append(chunk)
			block_id = self.upload_data_chunk(file_id, chunk_num, chunk)
			uploaded_chunk = self.unpack_block(self.get_block(block_id)['data'])[3]
			assert uploaded_chunk == chunk
			uploaded_chunks.append(uploaded_chunk)
			assert ''.join(uploaded_chunks) == data[:(chunk_num + 1) * offset_size]
			block_ids.append(block_id)
			chunk_num += 1
			if chunk_num == number_of_chunks:
				break
		assert ''.join(uploaded_chunks) == data
		return block_ids
	
	def download_data(self, block_ids):
		data = ''
		for block_id in block_ids:
			block = self.get_block(block_id)
			sections = self.unpack_block(block['data'])
			data += sections[3]
		return data
	
	def upload_file(self, filename):
		with open(filename, 'rb') as f:
			data = base64.b64encode(f.read()).decode('utf8')
		return self.upload_data(data)
	
	def download_file(self, block_ids, output_file):
		data = self.download_data(block_ids)
		with open(output_file, 'wb') as f:
			f.write(base64.b64decode(data))
	
	def create_account(self, set_credentials=None):
		out = self._call_bc('createAddress', {})
		res = out['result']
		keypair = (res['publicKey'], res['privateKey'])
		if set_credentials == True:
			self.set_credentials(keypair)
		return keypair

	def sign(self, message):
		# ok this is kinda nuts but the python secp was not working well..
		# so signing messages using a go program, using a temporary 
		# json file to pass messages. prob not a very secure option
		# but it will work for now. if anyone uses this, do something
		# better to protect the private keys.
		message_cb58 = cb58ref.cb58encode(message.encode('utf8'))
		tmpfile = './tmp'
		with open(tmpfile, 'w') as f:
			f.write(self.keypair[1] + '\n')
			f.write(message_cb58)
		os.system(f'go run keys.go sign {tmpfile}')
		with open(tmpfile) as f:
			sig = f.read()
		sig_len = str(len(sig))
		while len(sig_len) < 3:
			sig_len = '0' + sig_len
		while len(sig) < 100:
			sig += '\x00'
		os.remove(tmpfile)
		return self.keypair[0] + sig_len + sig + message
	
	def faucet(self, amount, recipient=None):
		if recipient is None: recipient = self.keypair[0]
		amt = str(amount)
		while len(amt) < 16:
			amt = '0' + amt
		if len(amt) > 16:
			raise Exception('not gonna work')
		data = amt + recipient
		payload = self.pack_block(9, data)
		return self.upload_block(payload)
	
	def get_unallocated_balance(self):
		return self._call_bc('getUnallocatedFunds', {})
	
	def transfer(self, amount, recipient):
		sender = self.keypair[0]
		amt = str(amount)
		while len(amt) < 16:
			amt = '0' + amt
		data = amt + sender + recipient
		payload = self.pack_block(1, data)
		return self.upload_block(payload)
	
	def stake(self, node_id, amount, start, end):
		sender = self.keypair[0]
		amt = str(amount)
		while len(amt) < 16:
			amt = '0' + amt
		start_str = str(int(start))
		while len(start_str) < 10:
			start_str = '0' + start_str
		end_str = str(int(end))
		while len(end_str) < 10:
			end_str = '0' + end_str

		if len(end_str) > 10 or len(start_str) > 10 or len(amt) > 16:
			raise Exception("input data incorrect")
		# node_id = 40 bytes
		data = node_id + sender + start_str + end_str + amt
		payload = self.pack_block(2, data)
		return self.upload_block(payload)
		
	def was_validating_at(self, node_id, timestamp):
		return self._call_bc('getValidatorsAt', {
			'timestamp': timestamp,
			'nodeID': node_id,
		})

	
api = FilestorageAPI('http://localhost:9658', sys.argv[1])
print("""

  ______ _____ _      ______  _____ _______ ____  _____            _____ ________      ____  __ 
 |  ____|_   _| |    |  ____|/ ____|__   __/ __ \|  __ \     /\   / ____|  ____\ \    / /  \/  |
 | |__    | | | |    | |__  | (___    | | | |  | | |__) |   /  \ | |  __| |__   \ \  / /| \  / |
 |  __|   | | | |    |  __|  \___ \   | | | |  | |  _  /   / /\ \| | |_ |  __|   \ \/ / | |\/| |
 | |     _| |_| |____| |____ ____) |  | | | |__| | | \ \  / ____ \ |__| | |____   \  /  | |  | |
 |_|    |_____|______|______|_____/   |_|  \____/|_|  \_\/_/    \_\_____|______|   \/   |_|  |_|
                                                                                                

				Welcome ! 

				You now have access to the api via the `api` variable.

				For docs, check out:
					https://github.com/connorbode/filestoragevm/blob/main/cli/README.md


""")

import ipdb; ipdb.set_trace()
