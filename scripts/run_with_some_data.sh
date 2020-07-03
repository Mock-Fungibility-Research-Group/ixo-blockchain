#!/usr/bin/env bash

PASSWORD="12345678"

ixod init local --chain-id pandora-1

yes $PASSWORD | ixocli keys delete miguel --force
yes $PASSWORD | ixocli keys add miguel

# Note: important to add 'miguel' as a genesis-account since this is the chain's validator
yes $PASSWORD | ixod add-genesis-account "$(ixocli keys show miguel -a)" 100000000uixos,100000000000uixo,1000000res,1000000rez

# Add pubkey-based genesis accounts
MIGUEL_ADDR="ixo107pmtx9wyndup8f9lgj6d7dnfq5kuf3sapg0vx"    # address from did:ixo:4XJLBfGtWSGKSz4BeRxdun's pubkey
yes $PASSWORD | ixod add-genesis-account "$MIGUEL_ADDR" 100000000uixos,100000000000uixo,1000000res,1000000rez

# Add genesis oracle
MIGUEL_DID="did:ixo:4XJLBfGtWSGKSz4BeRxdun"
yes $PASSWORD | ixod add-genesis-oracle "$MIGUEL_DID" "uixo:mint"

# Add ixo did
IXO_DID="did:ixo:U4tSpzzv91HHqWW1YmFkHJ"
FROM="\"ixo_did\": \"\""
TO="\"ixo_did\": \"$IXO_DID\""
sed -i "s/$FROM/$TO/" "$HOME"/.ixod/config/genesis.json

# Set staking token
STAKING_TOKEN="uixos"
FROM="\"bond_denom\": \"stake\""
TO="\"bond_denom\": \"$STAKING_TOKEN\""
sed -i "s/$FROM/$TO/" "$HOME"/.ixod/config/genesis.json

# Set min-gas-prices
FROM="minimum-gas-prices = \"\""
TO="minimum-gas-prices = \"\""
sed -i "s/$FROM/$TO/" "$HOME"/.ixod/config/app.toml

ixocli config chain-id pandora-1
ixocli config output json
ixocli config indent true
ixocli config trust-node true

yes $PASSWORD | ixod gentx --name miguel --amount 1000000uixos

ixod collect-gentxs
ixod validate-genesis

# Uncomment the below to broadcast node RPC endpoint
#FROM="laddr = \"tcp:\/\/127.0.0.1:26657\""
#TO="laddr = \"tcp:\/\/0.0.0.0:26657\""
#sed -i "s/$FROM/$TO/" "$HOME"/.ixod/config/config.toml

ixod start --pruning "syncable" &
ixocli rest-server --chain-id pandora-1 --trust-node && fg
