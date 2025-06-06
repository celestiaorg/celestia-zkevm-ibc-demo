#!/bin/bash
# SPDX-License-Identifier: MIT
#
# Copyright (c) 2024 Berachain Foundation
#
# Permission is hereby granted, free of charge, to any person
# obtaining a copy of this software and associated documentation
# files (the "Software"), to deal in the Software without
# restriction, including without limitation the rights to use,
# copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following
# conditions:
#
# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
# OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
# NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
# HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
# WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
# FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.

# function to resolve absolute path from relative
resolve_path() {
	if [[ "$1" =~ : ]]; then
        # treat as an address or url, return as is
        echo "$1"
	fi
    cd "$(dirname "$1")"
    local abs_path
    abs_path="$(pwd -P)/$(basename "$1")"
    echo "$abs_path"
}

CHAINID="beacond-2061"
MONIKER="localtestnet"
LOGLEVEL="info"
CONSENSUS_KEY_ALGO="bls12_381"
HOMEDIR="./.tmp/beacond"
RPC_PREFIX="http://"
RPC_DIAL_URL="reth:8551"


# Path variables
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json
ETH_GENESIS=$(resolve_path "./testing/files/eth-genesis.json")
echo "ETH_GENESIS: $ETH_GENESIS"
JWT_SECRET_PATH=$(resolve_path "./testing/files/jwt.hex")
echo "JWT_SECRET_PATH: $JWT_SECRET_PATH"
DA_AUTH_TOKEN=$(cat "./testing/files/da_auth_token")
echo "DA_AUTH_TOKEN: $DA_AUTH_TOKEN"

# used to exit on first error (any non-zero exit code)
set -e

# Reinstall daemon
make build

overwrite="Y"
# if [ -d $HOMEDIR ]; then
# 	printf "\nAn existing folder at '%s' was found. You can choose to delete this folder and start a new local node with new keys from genesis. When declined, the existing local node is started. \n" $HOMEDIR
# 	echo "Overwrite the existing configuration and start a new local node? [y/n]"
# 	read -r overwrite
# else
# overwrite="Y"
# fi

export CHAIN_SPEC="devnet"

# Setup local node if overwrite is set to Yes, otherwise skip setup
if [[ $overwrite == "y" || $overwrite == "Y" ]]; then
	rm -rf $HOMEDIR
	./build/bin/beacond init $MONIKER \
		--chain-id $CHAINID \
		--home $HOMEDIR \
		--consensus-key-algo $CONSENSUS_KEY_ALGO
	./build/bin/beacond genesis add-premined-deposit --home $HOMEDIR
	./build/bin/beacond genesis collect-premined-deposits --home $HOMEDIR
	./build/bin/beacond genesis execution-payload "$ETH_GENESIS" --home $HOMEDIR
fi

ADDRESS=$(jq -r '.address' $HOMEDIR/config/priv_validator_key.json)
PUB_KEY=$(jq -r '.pub_key' $HOMEDIR/config/priv_validator_key.json)
jq --argjson pubKey "$PUB_KEY" '.consensus["validators"]=[{"address": "'$ADDRESS'", "pub_key": $pubKey, "power": "32000000000", "name": "Rollkit Sequencer"}]' $GENESIS > temp.json && mv temp.json $GENESIS

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
BEACON_START_CMD="./build/bin/beacond start --pruning=nothing "$TRACE" \
--log_level $LOGLEVEL --api.enabled-unsafe-cors \
--rollkit.aggregator --rollkit.da_address http://celestia-network-bridge:26658 --rpc.laddr tcp://127.0.0.1:36657 --grpc.address 127.0.0.1:9290 --p2p.laddr "0.0.0.0:36656" \
--api.enable --api.swagger --minimum-gas-prices=0.0001abgt --rollkit.da_auth_token ${DA_AUTH_TOKEN} --rollkit.da_namespace 00000000000000000000000000000000000000b7b24d9321578eb83626 \
--home $HOMEDIR --beacon-kit.engine.jwt-secret-path ${JWT_SECRET_PATH} --rollkit.block_time 30s"

# Conditionally add the rpc-dial-url flag if RPC_DIAL_URL is not empty
if [ -n "$RPC_DIAL_URL" ]; then
	# this will overwrite the default dial url
	RPC_DIAL_URL=$(resolve_path "$RPC_DIAL_URL")
	echo "Overwriting the default dial url with $RPC_DIAL_URL"
	BEACON_START_CMD="$BEACON_START_CMD --beacon-kit.engine.rpc-dial-url ${RPC_PREFIX}${RPC_DIAL_URL}"
fi

echo $BEACON_START_CMD

# run the beacon node
eval $BEACON_START_CMD
