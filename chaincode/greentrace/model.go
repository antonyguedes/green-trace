package main

// ─────────────────────────────────────────────────────────────────────────────
// model.go — Estruturas de dados do Token de Conformidade Ambiental (TCA)
// Baseado em: Res. CMN 5.193/2024, 5.267/2025, 5.268/2025, BCB 140/2021
// ─────────────────────────────────────────────────────────────────────────────

// Status possíveis do TCA
const (
	StatusAtivo      = "ATIVO"
	StatusExpirado   = "EXPIRADO"
	StatusRevogado   = "REVOGADO"
	StatusSuspenso   = "SUSPENSO"
	StatusFinalizado = "FINALIZADO"
	StatusNegado     = "NEGADO"
)

// Validade padrão do TCA em meses — conforme Res. CMN 5.267/2025
// (fiscalização contínua durante vigência do financiamento)
const ValidadeMeses = 6

// Threshold mínimo de score para aprovação (critérios não-absolutos)
const ScoreMinimo = 80

// TCA — Token de Conformidade Ambiental
// Registrado no ledger do canal-conformidade
type TCA struct {
	// ── Identidade ──────────────────────────────────────────────────────────
	ID              string `json:"id"`              // ex: "TCA-CAR-SP-1234567-20260412"
	CodigoCAR       string `json:"codigoCAR"`       // identificador único no SICAR
	CPFCNPJHash     string `json:"cpfCnpjHash"`     // SHA-256 do CPF/CNPJ — nunca dado bruto
	InstFinEmissora string `json:"instFinEmissora"` // MSPID da IF solicitante

	// ── Ciclo de vida ────────────────────────────────────────────────────────
	Status             string      `json:"status"`             // ATIVO | EXPIRADO | REVOGADO | SUSPENSO | FINALIZADO | NEGADO
	DataEmissao        string      `json:"dataEmissao"`        // RFC3339
	DataValidade       string      `json:"dataValidade"`       // RFC3339 — emissao + 6 meses
	DataRevogacao      string      `json:"dataRevogacao"`      // preenchido só se REVOGADO
	MotivoRevogacao    string      `json:"motivoRevogacao"`    // descrição da revogação
	DataSuspensao      string      `json:"dataSuspensao"`      // preenchido se SUSPENSO
	MotivoSuspensao    string      `json:"motivoSuspensao"`    // descrição da suspensão atual
	DataFinalizacao    string      `json:"dataFinalizacao"`    // preenchido se FINALIZADO
	HistoricoSuspensoes []Suspensao `json:"historicoSuspensoes"`// armazena as suspensões e reativações passadas

	// ── Impedimentos absolutos ───────────────────────────────────────────────
	// Res. CMN 5.193/2024 e 5.268/2025: qualquer true bloqueia o TCA
	Impedimentos ImpedimentosAbsolutos `json:"impedimentos"`

	// ── Score complementar ───────────────────────────────────────────────────
	// Calculado apenas se todos os impedimentos absolutos forem false
	ScoreConformidade int  `json:"scoreConformidade"` // 0-100
	Aprovado          bool `json:"aprovado"`           // score >= ScoreMinimo

	// ── Cadeia de evidências ─────────────────────────────────────────────────
	Evidencias Evidencias `json:"evidencias"`

	// ── Reaproveitamento entre IFs ───────────────────────────────────────────
	TCAOrigemID string `json:"tcaOrigemID"` // vazio se verificação original
	RevalidadoEm string `json:"revalidadoEm"` // timestamp da revalidação
}

// ImpedimentosAbsolutos — critérios binários da Res. CMN 5.193/2024 e 5.268/2025
// Qualquer campo true = TCA não pode ser emitido
type ImpedimentosAbsolutos struct {
	EmbargoIBAMA       bool   `json:"embargoIBAMA"`       // embargo federal ativo
	EmbargoOEMA        bool   `json:"embargoOEMA"`        // embargo estadual ativo
	DesmatamentoIlegal bool   `json:"desmatamentoIlegal"` // PRODES/INPE — obrigatório em 2026
	TrabalhoEscravo    bool   `json:"trabalhoEscravo"`    // Cadastro MTE — BCB 140/2021
	FlorestaPublica    bool   `json:"florestaPublica"`    // área em floresta pública não destinada
	DetalheImpedimento string `json:"detalheImpedimento"` // descrição do impedimento encontrado
}

// Evidencias — cada elo da cadeia de evidências com fonte e timestamp
type Evidencias struct {
	CAR             Evidencia `json:"car"`
	IBAMA           Evidencia `json:"ibama"`
	OEMA            Evidencia `json:"oema"`
	INCRA           Evidencia `json:"incra"`
	PRODES          Evidencia `json:"prodes"`          // INPE — desmatamento
	ReservaLegal    Evidencia `json:"reservaLegal"`
	TrabalhoEscravo Evidencia `json:"trabalhoEscravo"` // Cadastro MTE
	FlorestaPublica Evidencia `json:"florestaPublica"` // Res. CMN 5.268/2025
}

// Evidencia — registro imutável de uma consulta a fonte oficial
type Evidencia struct {
	Fonte        string `json:"fonte"`        // ex: "SICAR", "IBAMA", "INPE/PRODES"
	Resultado    string `json:"resultado"`    // ex: "ATIVO", "LIMPO", "32%_averbada"
	Timestamp    string `json:"timestamp"`    // quando foi consultado (RFC3339)
	HashResposta string `json:"hashResposta"` // SHA-256 da resposta da API
	Conforme     bool   `json:"conforme"`     // true = este elo está ok
	PesoScore    int    `json:"pesoScore"`    // contribuição para o score total (0-20)
}

// SolicitacaoTCA — payload de entrada para EmitirTCA
type SolicitacaoTCA struct {
	CodigoCAR   string `json:"codigoCAR"`
	CPFCNPJHash string `json:"cpfCnpjHash"`
}

// RespostaConsulta — retorno de ConsultarTCA
type RespostaConsulta struct {
	Encontrado    bool   `json:"encontrado"`
	TCA           *TCA   `json:"tca,omitempty"`
	PodeReutilizar bool  `json:"podeReutilizar"` // true se válido e < 6 meses
	Mensagem      string `json:"mensagem"`
}

// Suspensao — armazena histórico de eventos de suspensão e reativação no ciclo de vida
type Suspensao struct {
	Motivo       string `json:"motivo"`
	DataSuspenso string `json:"dataSuspenso"`
	ReativadoEm  string `json:"reativadoEm"`
}

// HistoricoTCA — usado para visualização das transferências / log de auditoria
type HistoricoTCA struct {
	TxId      string `json:"txId"`
	Timestamp string `json:"timestamp"`
	TCA       *TCA   `json:"tca"`
}
