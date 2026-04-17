#!/bin/bash
set -e

# ─────────────────────────────────────────────────────────────────────────────
# upgrade_chaincode.sh — Atualização do Chaincode GreenTrace
# Executa: install → approve → commit
# Mostra a versão atual e solicita a nova versão ao usuário
# ─────────────────────────────────────────────────────────────────────────────

NETWORK_DIR="/home/antonioforte/Projects/green-trace/network"
CRYPTO="$NETWORK_DIR/crypto-config"
ORDERER_CA="$CRYPTO/ordererOrganizations/bancocentral.green-trace.com/tlsca/tlsca.bancocentral.green-trace.com-cert.pem"
POLICY="OR('BancoCentralMSP.peer','InstFinAMSP.peer','InstFinBMSP.peer','OrgAmbientalMSP.peer')"

export FABRIC_CFG_PATH="$NETWORK_DIR/fabric-samples/config"
export CORE_PEER_TLS_ENABLED=true

echo "═══════════════════════════════════════════"
echo "  GreenTrace — Atualização de Chaincode"
echo "═══════════════════════════════════════════"

# Configura o peer do Banco Central como padrão para buscar informações
export CORE_PEER_LOCALMSPID="BancoCentralMSP"
export CORE_PEER_ADDRESS="peer0.bancocentral.green-trace.com:7051"
export CORE_PEER_TLS_ROOTCERT_FILE="$CRYPTO/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt"
export CORE_PEER_MSPCONFIGPATH="$CRYPTO/peerOrganizations/bancocentral.green-trace.com/users/Admin@bancocentral.green-trace.com/msp"

echo ""
echo "▶ Buscando informações atuais do chaincode no canal..."
CURRENT_INFO=$(peer lifecycle chaincode querycommitted -C greentracechannel -n greentrace || true)

PAST_VERSION="Desconhecida"
PAST_SEQUENCE="0"

if [[ $CURRENT_INFO == *"Version:"* ]]; then
    # Extrai pegando tudo a partir de 'Version:' até a vírgula
    PAST_VERSION=$(echo "$CURRENT_INFO" | grep -o 'Version: [^,]*' | cut -d' ' -f2)
    PAST_SEQUENCE=$(echo "$CURRENT_INFO" | grep -o 'Sequence: [0-9]*' | cut -d' ' -f2)
    echo "  → Versão atual instalada  : $PAST_VERSION"
    echo "  → Sequência atual instalada: $PAST_SEQUENCE"
else
    echo "  → Atenção: Chaincode 'greentrace' não encontrado como comitado no canal ou falha ao buscar."
fi
echo ""

# Sugere a próxima sequência se conseguimos ler a atual
SUGGESTED_SEQUENCE=$((PAST_SEQUENCE + 1))

read -p "Digite a NOVA versão (ex: 1.1, 2.0. Atual é $PAST_VERSION): " NEW_VERSION
read -p "Digite a NOVA sequência (deve ser maior que $PAST_SEQUENCE. Sugestão: $SUGGESTED_SEQUENCE): " NEW_SEQUENCE

if [ -z "$NEW_VERSION" ] || [ -z "$NEW_SEQUENCE" ]; then
    echo "Erro: Versão ou sequência inválidas. Cancelando."
    exit 1
fi

echo ""
echo "▶ [1/4] Empacotando o novo chaincode..."
peer lifecycle chaincode package "$NETWORK_DIR/greentrace.tar.gz" \
  --path "$NETWORK_DIR/../chaincode/greentrace" \
  --lang golang \
  --label "greentrace_${NEW_VERSION}"

echo ""
echo "▶ [2/4] Install do novo pacote do chaincode..."

install_on_peer() {
  local MSPID=$1
  local ADDRESS=$2
  local TLSCERT=$3
  local MSPPATH=$4

  echo "  → Instalando pacote em $MSPID..."
  CORE_PEER_LOCALMSPID=$MSPID \
  CORE_PEER_ADDRESS=$ADDRESS \
  CORE_PEER_TLS_ROOTCERT_FILE=$TLSCERT \
  CORE_PEER_MSPCONFIGPATH=$MSPPATH \
  peer lifecycle chaincode install "$NETWORK_DIR/greentrace.tar.gz" > "log_install_${MSPID}.txt" 2>&1 || true
}

install_on_peer "BancoCentralMSP" "peer0.bancocentral.green-trace.com:7051" \
  "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/users/Admin@bancocentral.green-trace.com/msp"

install_on_peer "InstFinAMSP" "peer0.instfina.green-trace.com:8051" \
  "$CRYPTO/peerOrganizations/instfina.green-trace.com/peers/peer0.instfina.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/instfina.green-trace.com/users/Admin@instfina.green-trace.com/msp"

install_on_peer "InstFinBMSP" "peer0.instfinb.green-trace.com:9051" \
  "$CRYPTO/peerOrganizations/instfinb.green-trace.com/peers/peer0.instfinb.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/instfinb.green-trace.com/users/Admin@instfinb.green-trace.com/msp"

install_on_peer "OrgAmbientalMSP" "peer0.orgambiental.green-trace.com:10051" \
  "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/peers/peer0.orgambiental.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/users/Admin@orgambiental.green-trace.com/msp"

# Tenta calcular o Package ID diretamente do arquivo tar.gz usando a ferramenta do peer
PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid "$NETWORK_DIR/greentrace.tar.gz" 2>/dev/null || true)

if [ -z "$PACKAGE_ID" ]; then
    # Fallback para parsing do log de instalação caso 'calculatepackageid' dê pau
    PACKAGE_ID=$(grep 'Chaincode code package identifier:' log_install_BancoCentralMSP.txt | awk '{print $5}' || true)
fi

if [ -z "$PACKAGE_ID" ]; then
    echo "Erro: Não foi possível determinar o PACKAGE_ID."
    cat log_install_BancoCentralMSP.txt
    rm -f log_install_*.txt
    exit 1
fi

rm -f log_install_*.txt
echo "  → Package ID obtido: $PACKAGE_ID"

echo ""
echo "▶ [3/4] Approve do chaincode para a versão $NEW_VERSION..."

approve_for_org() {
  local MSPID=$1
  local ADDRESS=$2
  local TLSCERT=$3
  local MSPPATH=$4

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
    --name greentrace --version "$NEW_VERSION" \
    --package-id "$PACKAGE_ID" --sequence "$NEW_SEQUENCE" \
    --signature-policy "$POLICY"
  echo "  ✅ $MSPID aprovou"
}

approve_for_org "BancoCentralMSP" "peer0.bancocentral.green-trace.com:7051" \
  "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/bancocentral.green-trace.com/users/Admin@bancocentral.green-trace.com/msp"

approve_for_org "InstFinAMSP" "peer0.instfina.green-trace.com:8051" \
  "$CRYPTO/peerOrganizations/instfina.green-trace.com/peers/peer0.instfina.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/instfina.green-trace.com/users/Admin@instfina.green-trace.com/msp"

approve_for_org "InstFinBMSP" "peer0.instfinb.green-trace.com:9051" \
  "$CRYPTO/peerOrganizations/instfinb.green-trace.com/peers/peer0.instfinb.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/instfinb.green-trace.com/users/Admin@instfinb.green-trace.com/msp"

approve_for_org "OrgAmbientalMSP" "peer0.orgambiental.green-trace.com:10051" \
  "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/peers/peer0.orgambiental.green-trace.com/tls/ca.crt" \
  "$CRYPTO/peerOrganizations/orgambiental.green-trace.com/users/Admin@orgambiental.green-trace.com/msp"

echo ""
echo "▶ [4/4] Commit da nova versão ($NEW_VERSION / seq $NEW_SEQUENCE)..."

export CORE_PEER_LOCALMSPID=BancoCentralMSP
export CORE_PEER_ADDRESS=peer0.bancocentral.green-trace.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE="$CRYPTO/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt"
export CORE_PEER_MSPCONFIGPATH="$CRYPTO/peerOrganizations/bancocentral.green-trace.com/users/Admin@bancocentral.green-trace.com/msp"

peer lifecycle chaincode commit \
  -o orderer.bancocentral.green-trace.com:7050 \
  --ordererTLSHostnameOverride orderer.bancocentral.green-trace.com \
  --tls --cafile "$ORDERER_CA" \
  --channelID greentracechannel \
  --name greentrace --version "$NEW_VERSION" --sequence "$NEW_SEQUENCE" \
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
echo "  ✅ Upgrade concluído com sucesso!"
echo "  Chaincode 'greentrace' atualizado:"
echo "  Versão: $NEW_VERSION | Sequência: $NEW_SEQUENCE"
echo "═══════════════════════════════════════════"
