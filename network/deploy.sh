#!/bin/bash
set -e

# ─────────────────────────────────────────────────────────────────────────────
# deploy.sh — Deploy completo da rede GreenTrace
# Executa: osnadmin → join → install → approve → commit
# ─────────────────────────────────────────────────────────────────────────────

NETWORK_DIR="/home/antonioforte/Projects/green-trace/network"
CRYPTO="$NETWORK_DIR/crypto-config"
ORDERER_CA="$CRYPTO/ordererOrganizations/bancocentral.green-trace.com/tlsca/tlsca.bancocentral.green-trace.com-cert.pem"
ORDERER_CERT="$CRYPTO/ordererOrganizations/bancocentral.green-trace.com/orderers/orderer.bancocentral.green-trace.com/tls/server.crt"
ORDERER_KEY="$CRYPTO/ordererOrganizations/bancocentral.green-trace.com/orderers/orderer.bancocentral.green-trace.com/tls/server.key"
CHANNEL_BLOCK="$NETWORK_DIR/channel-artifacts/greentracechannel.block"
export FABRIC_CFG_PATH="$NETWORK_DIR/fabric-samples/config"
export CORE_PEER_TLS_ENABLED=true

PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid "$NETWORK_DIR/greentrace.tar.gz")
POLICY="OR('BancoCentralMSP.peer','InstFinAMSP.peer','InstFinBMSP.peer','OrgAmbientalMSP.peer')"

echo "═══════════════════════════════════════════"
echo "  GreenTrace — Deploy Script"
echo "═══════════════════════════════════════════"

# ── 1. osnadmin channel join ─────────────────────────────────────────────────
echo ""
echo "▶ [1/4] Registrando canal no orderer..."
# osnadmin channel join \
#   --channelID greentracechannel \
#   --config-block "$CHANNEL_BLOCK" \
#   --orderer-address orderer.bancocentral.green-trace.com:7053 \
#   --ca-file "$ORDERER_CA" \
#   --client-cert "$ORDERER_CERT" \
#   --client-key "$ORDERER_KEY"

sleep 3

# ── 2. Join dos peers ─────────────────────────────────────────────────────────
echo ""
echo "▶ [2/4] Join dos peers no canal..."

join_peer() {
  local MSPID=$1
  local ADDRESS=$2
  local TLSCERT=$3
  local MSPPATH=$4

  CORE_PEER_LOCALMSPID=$MSPID \
  CORE_PEER_ADDRESS=$ADDRESS \
  CORE_PEER_TLS_ROOTCERT_FILE=$TLSCERT \
  CORE_PEER_MSPCONFIGPATH=$MSPPATH \
#   peer channel join \
#     -b "$CHANNEL_BLOCK" \
#     --orderer orderer.bancocentral.green-trace.com:7050 \
#     --tls --cafile "$ORDERER_CA"
  echo "  ✅ $MSPID joined"
}

join_peer "BancoCentralMSP" \
  "peer0.bancocentral.green-trace.com:7051" \
  "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/users/Admin@bancocentral.green-trace.com/msp"

join_peer "InstFinAMSP" \
  "peer0.instfina.green-trace.com:8051" \
  "$CRYPTO/peerOrganizations/instfina.green-trace.com/peers/peer0.instfina.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/instfina.green-trace.com/users/Admin@instfina.green-trace.com/msp"

join_peer "InstFinBMSP" \
  "peer0.instfinb.green-trace.com:9051" \
  "$CRYPTO/peerOrganizations/instfinb.green-trace.com/peers/peer0.instfinb.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/instfinb.green-trace.com/users/Admin@instfinb.green-trace.com/msp"

join_peer "OrgAmbientalMSP" \
  "peer0.orgambiental.green-trace.com:10051" \
  "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/peers/peer0.orgambiental.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/users/Admin@orgambiental.green-trace.com/msp"

# ── 3. Install + Approve ──────────────────────────────────────────────────────
echo ""
echo "▶ [3/4] Install e Approve do chaincode..."

install_and_approve() {
  local MSPID=$1
  local ADDRESS=$2
  local TLSCERT=$3
  local MSPPATH=$4

  echo "  → Instalando em $MSPID..."
  CORE_PEER_LOCALMSPID=$MSPID \
  CORE_PEER_ADDRESS=$ADDRESS \
  CORE_PEER_TLS_ROOTCERT_FILE=$TLSCERT \
  CORE_PEER_MSPCONFIGPATH=$MSPPATH \
  peer lifecycle chaincode install "$NETWORK_DIR/greentrace.tar.gz"

  echo "  → Aprovando para $MSPID..."
  CORE_PEER_LOCALMSPID=$MSPID \
  CORE_PEER_ADDRESS=$ADDRESS \
  CORE_PEER_TLS_ROOTCERT_FILE=$TLSCERT \
  CORE_PEER_MSPCONFIGPATH=$MSPPATH \
  peer lifecycle chaincode approveformyorg \
    -o orderer.bancocentral.green-trace.com:7050 \
    --ordererTLSHostnameOverride orderer.bancocentral.green-trace.com \
    --tls --cafile "$ORDERER_CA" \
    --channelID greentracechannel \
    --name greentrace --version 2.0 \
    --package-id "$PACKAGE_ID" --sequence 2 \
    --signature-policy "$POLICY"
  echo "  ✅ $MSPID aprovado"
}

install_and_approve "BancoCentralMSP" \
  "peer0.bancocentral.green-trace.com:7051" \
  "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/users/Admin@bancocentral.green-trace.com/msp"

install_and_approve "InstFinAMSP" \
  "peer0.instfina.green-trace.com:8051" \
  "$CRYPTO/peerOrganizations/instfina.green-trace.com/peers/peer0.instfina.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/instfina.green-trace.com/users/Admin@instfina.green-trace.com/msp"

install_and_approve "InstFinBMSP" \
  "peer0.instfinb.green-trace.com:9051" \
  "$CRYPTO/peerOrganizations/instfinb.green-trace.com/peers/peer0.instfinb.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/instfinb.green-trace.com/users/Admin@instfinb.green-trace.com/msp"

install_and_approve "OrgAmbientalMSP" \
  "peer0.orgambiental.green-trace.com:10051" \
  "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/peers/peer0.orgambiental.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/users/Admin@orgambiental.green-trace.com/msp"

# ── 4. Commit ─────────────────────────────────────────────────────────────────
echo ""
echo "▶ [4/4] Commit do chaincode..."

export CORE_PEER_LOCALMSPID=BancoCentralMSP
export CORE_PEER_ADDRESS=peer0.bancocentral.green-trace.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE="$CRYPTO/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt"
export CORE_PEER_MSPCONFIGPATH="$CRYPTO/peerOrganizations/bancocentral.green-trace.com/users/Admin@bancocentral.green-trace.com/msp"

peer lifecycle chaincode commit \
  -o orderer.bancocentral.green-trace.com:7050 \
  --ordererTLSHostnameOverride orderer.bancocentral.green-trace.com \
  --tls --cafile "$ORDERER_CA" \
  --channelID greentracechannel \
  --name greentrace --version 2.0 --sequence 2 \
  --signature-policy "$POLICY" \
  --peerAddresses peer0.bancocentral.green-trace.com:7051 \
  --tlsRootCertFiles "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt" \
  --peerAddresses peer0.instfina.green-trace.com:8051 \
  --tlsRootCertFiles "$CRYPTO/peerOrganizations/instfina.green-trace.com/peers/peer0.instfina.green-trace.com/tls/ca.crt" \
  --peerAddresses peer0.instfinb.green-trace.com:9051 \
  --tlsRootCertFiles "$CRYPTO/peerOrganizations/instfinb.green-trace.com/peers/peer0.instfinb.green-trace.com/tls/ca.crt" \
  --peerAddresses peer0.orgambiental.green-trace.com:10051 \
  --tlsRootCertFiles "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/peers/peer0.orgambiental.green-trace.com/tls/ca.crt"

echo ""
echo "═══════════════════════════════════════════"
echo "  ✅ Deploy concluído com sucesso!"
echo "  Chaincode 'greentrace' ativo no canal"
echo "  'greentracechannel'"
echo "═══════════════════════════════════════════"