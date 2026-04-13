# 🌿 GreenTrace

> **Blockchain-based Environmental Compliance for Rural Credit in Brazil**
---

## Overview

GreenTrace is a permissioned blockchain network that automates the issuance of **Environmental Compliance Tokens (TCA — Token de Conformidade Ambiental)** for rural credit operations in Brazil.

Today, financial institutions verify environmental compliance manually — consulting multiple government systems, generating PDFs, and repeating the process for every credit operation. GreenTrace replaces this fragmented workflow with an **immutable, auditable, and shareable evidence chain** on a Hyperledger Fabric network.

Each TCA encodes a complete chain of evidence from official Brazilian environmental sources, verifying that a rural property complies with the requirements of **Resoluções CMN nº 5.193/2024, 5.267/2025, and 5.268/2025** before credit is granted.

---

## Architecture

### Network Structure

```
┌─────────────────────────────────────────────────────────┐
│                  GREENTRACE NETWORK                     │
│                                                         │
│  Org1: BancoCentral  (regulator / auditor)              │
│  Org2: InstFinA      (financial institution A)          │
│  Org3: InstFinB      (financial institution B)          │
│  Org4: OrgAmbiental  (environmental data oracle)        │
│                                                         │
│  Orderer: OrdererBC  (Raft consensus)                   │
│  Channel: greentracechannel                             │
│  State DB: CouchDB   (rich query support)               │
└─────────────────────────────────────────────────────────┘
```

### Tech Stack

| Component | Technology |
|---|---|
| Blockchain | Hyperledger Fabric 3.1.4 |
| Consensus | etcd/Raft |
| Chaincode | Go 1.21 + fabric-contract-api-go |
| State Database | CouchDB 3.3.3 |
| Infrastructure | Docker Compose |
| Certificate Authority | cryptogen (dev) |

---

## The TCA — Environmental Compliance Token

A TCA is a JSON document recorded immutably on the ledger. It contains:

- **Identity**: CAR code (Brazil's Rural Environmental Registry), hashed CPF/CNPJ, issuing institution
- **Lifecycle**: Status (`ATIVO` / `EXPIRADO` / `REVOGADO`), issuance date, 6-month validity
- **Absolute Impediments**: Binary flags per CMN 5.193/2024 — any `true` blocks token issuance
- **Compliance Score**: 0–100 score from non-absolute criteria (threshold: 80)
- **Evidence Chain**: One entry per official data source, each with timestamp and SHA-256 hash

### Evidence Sources

| Source | Data | Basis |
|---|---|---|
| SICAR/MMA | CAR active status, no overlap with protected areas | Código Florestal |
| IBAMA | Federal embargo status | CMN 5.193/2024 |
| OEMA | State-level embargo status | CMN 5.268/2025 |
| INCRA/SNCR | Land tenure regularity | MCR |
| INPE/PRODES | Illegal deforestation detection | CMN 5.193/2024 (mandatory 2026) |
| SICAR/CAR | Legal Reserve percentage and registration | Código Florestal |
| MTE | Slave labor employer registry | BCB 140/2021 |
| SFB | Public forest non-overlap | CMN 5.268/2025 |

### TCA Lifecycle

```
EmitirTCA ──► ATIVO ──► ExpirarTCAs ──► EXPIRADO
                │
                └──► RevogarTCA ──────► REVOGADO

ATIVO ──► ConsultarTCA ──► RevalidarTCA ──► new ATIVO (linked to new IF)
```

---

## Chaincode Functions

| Function | Access | Description |
|---|---|---|
| `EmitirTCA` | Any IF | Queries all environmental sources and issues a TCA if compliant |
| `ConsultarTCA` | Any org | Returns the latest active TCA for a given CAR code |
| `RevalidarTCA` | Any IF | Reuses static evidence, re-validates dynamic data (IBAMA, PRODES) |
| `RevogarTCA` | BC or issuing IF | Revokes an active TCA with a stated reason |
| `ListarTCAs` | BancoCentralMSP only | Returns all TCAs for regulatory oversight |
| `ExpirarTCAs` | Scheduler trigger | Marks expired TCAs as `EXPIRADO` |

---

## Prerequisites

- Docker & Docker Compose
- Hyperledger Fabric binaries 3.1.4 (`peer`, `configtxgen`, `cryptogen`, `osnadmin`)
- Go 1.21+
- Fabric samples (for `core.yaml` config)

Add peer hostnames to `/etc/hosts`:

```
127.0.0.1 peer0.bancocentral.green-trace.com
127.0.0.1 peer0.instfina.green-trace.com
127.0.0.1 peer0.instfinb.green-trace.com
127.0.0.1 peer0.orgambiental.green-trace.com
127.0.0.1 orderer.bancocentral.green-trace.com
```

---

## Project Structure

```
green-trace/
├── network/
│   ├── configtx.yaml              # Channel and org policies
│   ├── crypto-config.yaml         # Identity generation config
│   ├── docker-compose.yaml        # 4 peers + 4 CouchDBs + orderer + CLI
│   ├── deploy.sh                  # One-shot deploy script
│   ├── channel-artifacts/
│   │   └── greentracechannel.block
│   └── crypto-config/             # Generated certificates and keys
│       ├── ordererOrganizations/
│       └── peerOrganizations/
└── chaincode/
    └── greentrace/
        ├── main.go                # Chaincode entrypoint and contract functions
        ├── model.go               # TCA, Evidencias, Impedimentos data structures
        ├── sources.go             # Environmental source queries and scoring
        └── go.mod
```

---

## Getting Started

### 1. Generate Certificates and Channel Block

```bash
cd network/

# Generate crypto material
cryptogen generate --config=crypto-config.yaml

# Generate channel genesis block
configtxgen \
  -configPath . \
  -profile GreenTraceGenesis \
  -channelID greentracechannel \
  -outputBlock ./channel-artifacts/greentracechannel.block
```

### 2. Package the Chaincode

```bash
export FABRIC_CFG_PATH=./fabric-samples/config

peer lifecycle chaincode package greentrace.tar.gz \
  --path /absolute/path/to/chaincode/greentrace \
  --lang golang \
  --label greentrace_1.0
```

### 3. Start the Network and Deploy

```bash
# Start all containers
docker-compose up -d

# Wait for peers to initialize (~20s), then run the full deploy
sleep 20
bash deploy.sh
```

The `deploy.sh` script handles everything:
- Registers the channel with the orderer (`osnadmin`)
- Joins all 4 peers to the channel
- Installs, approves, and commits the chaincode

### 4. Issue Your First TCA

```bash
docker exec -it cli bash

export ORDERER_CA=/opt/crypto/ordererOrganizations/bancocentral.green-trace.com/tlsca/tlsca.bancocentral.green-trace.com-cert.pem

peer chaincode invoke \
  -o orderer.bancocentral.green-trace.com:7050 \
  --ordererTLSHostnameOverride orderer.bancocentral.green-trace.com \
  --tls --cafile $ORDERER_CA \
  -C greentracechannel -n greentrace \
  --peerAddresses peer0.bancocentral.green-trace.com:7051 \
  --tlsRootCertFiles /opt/crypto/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt \
  -c '{"function":"EmitirTCA","Args":["CAR-SP-1234567-2026","<sha256-of-cpf-cnpj>"]}'
```

### 5. Query a TCA

```bash
peer chaincode query \
  -C greentracechannel -n greentrace \
  -c '{"function":"ConsultarTCA","Args":["CAR-SP-1234567-2026"]}'
```

---

## Regulatory Basis

GreenTrace is designed to operationalize the following Brazilian regulations:

- **Resolução CMN nº 5.193/2024** — Mandatory environmental verification (CAR × PRODES × embargos) before rural credit
- **Resolução CMN nº 5.267/2025** — Continuous monitoring (documental, presencial, sensoriamento remoto) throughout the credit lifecycle
- **Resolução CMN nº 5.268/2025** — Updated socio-environmental impediments, including public forest non-overlap
- **Resolução BCB nº 140/2021** — Slave labor employer registry as credit impediment
- **Resolução CMN nº 4.945/2021** — PRSAC (Social, Environmental and Climate Responsibility Policy)

---

## Cross-Institution TCA Reuse

A key innovation: when Institution B wants to verify a property already checked by Institution A, GreenTrace does not repeat the full verification. Instead:

1. Static evidence (CAR, INCRA, Legal Reserve) is **reused** from the original TCA
2. Dynamic impediments (IBAMA embargos, PRODES deforestation) are **always re-validated** — these change frequently
3. A **new TCA is issued** linked to Institution B, with full audit trail back to the origin

This reduces compliance cost across the system while maintaining regulatory integrity.

---

## CouchDB Dashboard

Each peer has its own CouchDB instance. You can inspect the ledger state via the Fauxton UI:

| Org | CouchDB URL |
|---|---|
| BancoCentral | http://localhost:5984/_utils |
| InstFinA | http://localhost:6984/_utils |
| InstFinB | http://localhost:7984/_utils |
| OrgAmbiental | http://localhost:8984/_utils |

Credentials: `admin` / `adminpw`

---

## Roadmap

- [ ] Connect to live Brazilian government APIs (SICAR, IBAMA, PRODES)
- [ ] Implement W3C Verifiable Credentials for producer identity (Hyperledger Aries)
- [ ] Build REST API gateway (Fabric Gateway SDK)
- [ ] Public dashboard with aggregated compliance metrics for BC regulatory oversight

---

## License

MIT License — see [LICENSE](LICENSE) for details.