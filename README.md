# 🌿 GreenTrace

> **Blockchain-based Environmental Compliance for Rural Credit in Brazil**

GreenTrace é uma plataforma descentralizada de conformidade ambiental que automatiza a emissão de **Tokens de Conformidade Ambiental (TCA)** para operações de crédito rural no Brasil, utilizando o **Hyperledger Fabric** como ledger imutável de evidências.

---

## O Problema

Instituições financeiras verificam a conformidade ambiental de propriedades rurais **manualmente** — consultando SICAR, IBAMA, PRODES, INCRA e outros sistemas, gerando documentos em PDF e repetindo o processo a cada operação. Esse fluxo é fragmentado, custoso e sujeito a fraude.

**GreenTrace substitui esse workflow** por uma cadeia de evidências imutável, auditável e compartilhável entre instituições — ancorada em blockchain permissionada.

---

## Arquitetura

GreenTrace é uma aplicação dApp full-stack em três camadas:

```
┌────────────────────────────────────────────────────────────────────────┐
│              FRONTEND  —  React 19 + Vite + TanStack                   │
│   Dashboard de Analista (IF) e Auditor (Banco Central)                 │
└─────────────────────────────┬──────────────────────────────────────────┘
                              │ REST (HTTP/JSON)
┌─────────────────────────────▼──────────────────────────────────────────┐
│      API GATEWAY  —  NestJS + Fabric Gateway SDK (Node.js)             │
│   Módulos: Fabric | TCA | Mock Oracle                                  │
└─────────────────────────────┬──────────────────────────────────────────┘
                              │ gRPC (TLS)
┌─────────────────────────────▼──────────────────────────────────────────┐
│       HYPERLEDGER FABRIC 3.1.4 — greentracechannel                     │
│                                                                        │
│   Org 1: BancoCentral    (regulador / auditor)      → peer :7051       │
│   Org 2: InstFinA        (Instituição Financeira A) → peer :8051       │
│   Org 3: InstFinB        (Instituição Financeira B) → peer :9051       │
│   Org 4: OrgAmbiental    (Oráculo de dados)         → peer :10051      │
│                                                                        │
│   Orderer: OrdererBC (Raft)    CouchDB por peer                        │
└────────────────────────────────────────────────────────────────────────┘
```

### Stack Tecnológico

| Componente | Tecnologia |
|---|---|
| Blockchain | Hyperledger Fabric 3.1.4 |
| Consenso | etcd / Raft |
| Chaincode | Go 1.25+ (`fabric-contract-api-go`) |
| State Database | CouchDB 3.3.3 _(rich queries)_ |
| API Gateway | NestJS 11 + `@hyperledger/fabric-gateway` |
| Frontend | React 19 + Vite + TanStack Router/Query |
| Estilo | Vanilla CSS (Glassmorphism / Dark Mode) |
| Identidade _(Fase 4)_ | W3C Verifiable Credentials (SSI / ZKP) |

---

## O Token de Conformidade Ambiental (TCA)

### Modelo de Dados Principal

```go
type TCA struct {
    ID              string // ex: "TCA-CAR-SP-1234567-1744912800"
    CodigoCAR       string // Identificador no SICAR
    CPFCNPJHash     string // SHA-256 — nunca dado bruto no ledger
    InstFinEmissora string // MSPID da IF (ex: "InstFinAMSP")

    // Lifecycle
    Status              string     // ATIVO | EXPIRADO | REVOGADO | SUSPENSO | FINALIZADO | NEGADO
    DataEmissao         string     // RFC3339
    DataValidade        string     // Emissão + 6 meses (Res. CMN 5.267/2025)
    DataSuspensao       string
    MotivoSuspensao     string
    DataFinalizacao     string
    HistoricoSuspensoes []Suspensao

    // Conformidade
    Impedimentos         ImpedimentosAbsolutos
    ScoreConformidade    int  // 0–100 (aprovado se >= 80)
    Aprovado             bool

    // Evidências encadeadas
    Evidencias Evidencias // Uma entrada por fonte oficial (hash SHA-256 + timestamp)

    // Reaproveitamento multi-IF
    TCAOrigemID string
    RevalidadoEm string
}
```

### Fontes de Evidência

| Fonte | Dado Verificado | Base Legal |
|---|---|---|
| SICAR/MMA | CAR ativo, sem sobreposição | Código Florestal |
| IBAMA | Embargos federais | Res. CMN 5.193/2024 |
| OEMA | Embargos estaduais | Res. CMN 5.268/2025 |
| INCRA/SNCR | Situação fundiária | MCR |
| INPE/PRODES | Desmatamento ilegal | Res. CMN 5.193/2024 _(obrigatório 2026)_ |
| SICAR/CAR | Reserva Legal averbada | Código Florestal |
| MTE | Cadastro de trabalho escravo | Res. BCB 140/2021 |
| SFB | Floresta pública não destinada | Res. CMN 5.268/2025 |

### Ciclo de Vida

```
EmitirTCA ──► ATIVO ────► SuspenderTCA ────► SUSPENSO
                │              │                  │
                │              │                  └─► ReativarTCA ──► ATIVO
                │              │
                │              └────────────────────► RevogarTCA ───► REVOGADO
                │
                ├──────────────────────────────────► FinalizarTCA ──► FINALIZADO
                │
                └──────────────────────────────────► RevalidarTCA ──► new ATIVO (nova IF)
```

**Impedimentos absolutos** (qualquer `true` bloqueia a emissão → status `NEGADO`):
- Embargo IBAMA ativo
- Embargo OEMA (estadual) ativo
- Desmatamento ilegal detectado pelo INPE/PRODES
- Produtor no Cadastro de Empregadores MTE (trabalho escravo)
- Imóvel em floresta pública não destinada

---

## Chaincode — Funções Disponíveis

| Função | Acesso Permitido | Descrição |
|---|---|---|
| `EmitirTCA` | Qualquer IF | Valida evidências e registra o TCA |
| `ConsultarTCA` | Qualquer org | Retorna o TCA atual por CAR |
| `RevalidarTCA` | Qualquer IF | Revalida com novas evidências dinâmicas |
| `SuspenderTCA` | BC ou IF emissora | Suspende por infração detectada |
| `ReativarTCA` | BC ou IF emissora | Reativa após regularização comprovada |
| `FinalizarTCA` | BC ou IF emissora | Finaliza após quitação do crédito |
| `RevogarTCA` | BC ou IF emissora | Revogação definitiva com motivo |
| `ConsultarMeusTCAs` | IF autenticada | Lista TCAs emitidos pela própria IF |
| `ListarTCAs` | `BancoCentralMSP` | Visão regulatória global |
| `AuditarTransacoes` | `BancoCentralMSP` | Histórico completo de estados (ledger) |

---

## API Gateway — Endpoints REST

Swagger disponível em `http://localhost:3000/api/docs`.

A identidade da organização é passada via header `x-organization` (ex: `BancoCentral`, `InstFinA`).

```
POST   /tca                      # Emitir novo TCA (Oráculo é consultado automaticamente)
GET    /tca/meus                  # Listar TCAs da minha IF
GET    /tca/todos                 # Visão BC: todos os TCAs do ledger
GET    /tca/:codigoCAR            # Consultar estado atual
GET    /tca/:codigoCAR/historico  # Auditar histórico de transações
POST   /tca/revalidar             # Revalidar com novas evidências
POST   /tca/:id/suspender         # Suspender
POST   /tca/:id/reativar          # Reativar
POST   /tca/:id/finalizar         # Finalizar
POST   /tca/:id/revogar           # Revogar
```

---

## Estrutura do Projeto

```
green-trace/
├── chaincode/
│   └── greentrace/
│       ├── main.go          # Funções do contrato e lógica do ciclo de vida
│       ├── model.go         # Structs: TCA, Evidencias, Impedimentos, Suspensao
│       ├── sources.go       # Avaliação de impedimentos, score e mock oracle
│       └── go.mod
│
├── network/
│   ├── configtx.yaml        # Políticas de canal e organizações
│   ├── crypto-config.yaml   # Geração de identidades criptográficas
│   ├── docker-compose.yaml  # 4 peers + 4 CouchDBs + orderer + CLI
│   ├── deploy.sh            # Deploy inicial completo (join → install → commit)
│   ├── upgrade_chaincode.sh # Upgrade inteligente com empacotamento automático
│   ├── channel-artifacts/
│   │   └── greentracechannel.block
│   └── crypto-config/       # Certificados e chaves gerados (não comitar)
│
├── api/
│   └── src/
│       ├── fabric/          # ConnectionService (gRPC) + ContractService (wrapper)
│       ├── tca/             # TcaController + TcaModule
│       ├── oracle/          # Mock Oracle: simula SICAR/IBAMA/PRODES/MTE
│       └── main.ts          # Bootstrap NestJS + Swagger + CORS
│
└── frontend/
    └── src/
        ├── api.ts           # Axios client com interceptor de identidade
        ├── App.tsx          # Dashboard principal + seletor de organização
        └── components/
            ├── TcaCard.tsx  # Card de exibição do TCA com ações
            └── ScoreBadge.tsx # Indicador visual de score de conformidade
```

---

## Pré-requisitos

- **Docker** e **Docker Compose**
- **Hyperledger Fabric Binaries 3.1.4** (`peer`, `configtxgen`, `cryptogen`, `osnadmin`) no PATH
- **Node.js 22+** e npm
- **Go 1.25+**

Adicione os hosts ao `/etc/hosts`:

```
127.0.0.1  peer0.bancocentral.green-trace.com
127.0.0.1  peer0.instfina.green-trace.com
127.0.0.1  peer0.instfinb.green-trace.com
127.0.0.1  peer0.orgambiental.green-trace.com
127.0.0.1  orderer.bancocentral.green-trace.com
```

---

## Como Executar

### 1. Subir a rede Fabric

```bash
cd network
cryptogen generate --config=crypto-config.yaml
configtxgen -configPath . -profile GreenTraceGenesis \
  -channelID greentracechannel -outputBlock ./channel-artifacts/greentracechannel.block

docker compose up -d
sleep 20
bash deploy.sh
```

### 2. Fazer Upgrade do Chaincode (após mudanças)

```bash
cd network
bash upgrade_chaincode.sh
# → informe a nova versão (ex: 4.0) e sequência (ex: 4)
```

### 3. Iniciar a API Gateway

```bash
cd api
npm install
npm run start:dev
# Acesse: http://localhost:3000/api/docs
```

### 4. Iniciar o Frontend

```bash
cd frontend
npm install
npm run dev
# Acesse: http://localhost:5173
```

---

## CouchDB (Inspeção do Ledger)

| Organização | URL |
|---|---|
| BancoCentral | http://localhost:5984/_utils |
| InstFinA | http://localhost:6984/_utils |
| InstFinB | http://localhost:7984/_utils |
| OrgAmbiental | http://localhost:8984/_utils |

Credenciais: `admin` / `adminpw`

---

## Base Regulatória

| Norma | Objeto |
|---|---|
| Res. CMN nº 5.193/2024 | Verificação ambiental obrigatória antes do crédito rural |
| Res. CMN nº 5.267/2025 | Monitoramento contínuo (documental, presencial e remoto) |
| Res. CMN nº 5.268/2025 | Impedimentos socioambientais atualizados (floresta pública) |
| Res. BCB nº 140/2021 | Cadastro de trabalho escravo como impedimento ao crédito |
| Res. CMN nº 4.945/2021 | PRSAC — Política de Responsabilidade Socioambiental |

---

## Roadmap

- [x] Rede Hyperledger Fabric 3.1.4 com 4 organizações
- [x] Chaincode Go com ciclo de vida estendido (`SUSPENSO`, `FINALIZADO`, `NEGADO`)
- [x] Auditoria completa via `GetHistoryForKey`
- [x] Reaproveitamento de TCA com revalidação de impedimentos dinâmicos
- [x] API Gateway NestJS com Fabric SDK (gRPC nativo, identidades por org)
- [x] Mock Oracle integrado (SICAR / IBAMA / INPE / MTE)
- [x] Dashboard React 19 com TanStack Query + Design System Glassmorphism
- [ ] **Fase 4**: W3C Verifiable Credentials para identidade do produtor (SSI / ZKP)
- [ ] Integração real com APIs governamentais (TRL 7+)

---

## Licença

MIT License — veja [LICENSE](LICENSE) para detalhes.