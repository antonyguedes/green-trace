package main

// ─────────────────────────────────────────────────────────────────────────────
// sources.go — Consulta às fontes ambientais oficiais
//
// No protótipo do LIFT Lab, estas funções simulam as respostas das APIs.
// Na versão de produção (TRL 7+), cada função faz uma chamada HTTP real
// a um serviço intermediário que consulta:
//   - SICAR (CAR): https://consultapublica.car.gov.br/publico/imoveis/index
//   - IBAMA: https://servicos.ibama.gov.br/ctf/publico/areasembargadas/
//   - INPE/PRODES: https://terrabrasilis.dpi.inpe.br/geoserver/
//   - INCRA: https://sncr.serpro.gov.br/
//   - MTE (trabalho escravo): https://www.gov.br/trabalho/cadastro-de-empregadores
//
// O hash da resposta garante a integridade da evidência no ledger.
// ─────────────────────────────────────────────────────────────────────────────

import (
	"fmt"
	"time"
)

// ResultadoRevalidacao — retorno da revalidação de impedimentos dinâmicos
type ResultadoRevalidacao struct {
	EvidenciaIBAMA  Evidencia
	EvidenciaOEMA   Evidencia
	EvidenciaPRODES Evidencia
}

// consultarFontesAmbientais — consulta todas as fontes e retorna as evidências
// DEPRECATED — Agora recebido via API para garantir determinismo no consenso Fabric (evitando time.Now() e chamadas externas no chaincode)
func consultarFontesAmbientais(codigoCAR string) (Evidencias, error) {
	agora := time.Now().UTC().Format(time.RFC3339)

	// ── 1. CAR / SICAR ───────────────────────────────────────────────────────
	// Verifica se o imóvel tem CAR ativo e sem sobreposição com áreas protegidas
	resCAR := fmt.Sprintf("CAR:%s:ATIVO:SEM_SOBREPOSICAO", codigoCAR)
	car := Evidencia{
		Fonte:        "SICAR/MMA",
		Resultado:    "ATIVO",
		Timestamp:    agora,
		HashResposta: gerarHash(resCAR),
		Conforme:     true,
		PesoScore:    20,
	}

	// ── 2. IBAMA — embargos federais ─────────────────────────────────────────
	resIBAMA := fmt.Sprintf("IBAMA:%s:LIMPO", codigoCAR)
	ibama := Evidencia{
		Fonte:        "IBAMA",
		Resultado:    "SEM_EMBARGO",
		Timestamp:    agora,
		HashResposta: gerarHash(resIBAMA),
		Conforme:     true,
		PesoScore:    0, // impedimento absoluto — não contribui para score
	}

	// ── 3. OEMA — embargos estaduais ─────────────────────────────────────────
	resOEMA := fmt.Sprintf("OEMA:%s:LIMPO", codigoCAR)
	oema := Evidencia{
		Fonte:        "OEMA",
		Resultado:    "SEM_EMBARGO_ESTADUAL",
		Timestamp:    agora,
		HashResposta: gerarHash(resOEMA),
		Conforme:     true,
		PesoScore:    0, // impedimento absoluto
	}

	// ── 4. INCRA — situação fundiária ────────────────────────────────────────
	resINCRA := fmt.Sprintf("INCRA:%s:REGULAR", codigoCAR)
	incra := Evidencia{
		Fonte:        "INCRA/SNCR",
		Resultado:    "SITUACAO_REGULAR",
		Timestamp:    agora,
		HashResposta: gerarHash(resINCRA),
		Conforme:     true,
		PesoScore:    20,
	}

	// ── 5. PRODES/INPE — desmatamento ilegal ─────────────────────────────────
	// Obrigatório a partir de 2026 — Res. CMN 5.193/2024
	resPRODES := fmt.Sprintf("PRODES:%s:SEM_DESMATAMENTO_2008_2026", codigoCAR)
	prodes := Evidencia{
		Fonte:        "INPE/PRODES",
		Resultado:    "SEM_DESMATAMENTO_ILEGAL",
		Timestamp:    agora,
		HashResposta: gerarHash(resPRODES),
		Conforme:     true,
		PesoScore:    0, // impedimento absoluto
	}

	// ── 6. Reserva Legal ─────────────────────────────────────────────────────
	// Percentual mínimo por bioma: Amazônia 80%, Cerrado 35%, demais 20%
	resRL := fmt.Sprintf("RL:%s:32PCT_AVERBADA", codigoCAR)
	reservaLegal := Evidencia{
		Fonte:        "SICAR/CAR",
		Resultado:    "32%_AVERBADA",
		Timestamp:    agora,
		HashResposta: gerarHash(resRL),
		Conforme:     true,
		PesoScore:    30,
	}

	// ── 7. Trabalho escravo — Cadastro MTE ───────────────────────────────────
	// Res. BCB 140/2021: impedem acesso ao crédito rural
	resMTE := fmt.Sprintf("MTE:%s:LIMPO", codigoCAR)
	trabalhoEscravo := Evidencia{
		Fonte:        "MTE/Cadastro_Empregadores",
		Resultado:    "SEM_RESTRICAO",
		Timestamp:    agora,
		HashResposta: gerarHash(resMTE),
		Conforme:     true,
		PesoScore:    0, // impedimento absoluto
	}

	// ── 8. Floresta pública — Res. CMN 5.268/2025 ────────────────────────────
	resFP := fmt.Sprintf("FLORESTA_PUBLICA:%s:NAO_INCIDE", codigoCAR)
	florestaPublica := Evidencia{
		Fonte:        "SFB/Cadastro_Florestas_Publicas",
		Resultado:    "NAO_INCIDE_FLORESTA_PUBLICA",
		Timestamp:    agora,
		HashResposta: gerarHash(resFP),
		Conforme:     true,
		PesoScore:    0, // impedimento absoluto
	}

	return Evidencias{
		CAR:             car,
		IBAMA:           ibama,
		OEMA:            oema,
		INCRA:           incra,
		PRODES:          prodes,
		ReservaLegal:    reservaLegal,
		TrabalhoEscravo: trabalhoEscravo,
		FlorestaPublica: florestaPublica,
	}, nil
}

// revalidarImpedimentosDinamicos — revalida apenas IBAMA, OEMA e PRODES
// (dados que mudam com frequência — usados no RevalidarTCA)
// DEPRECATED — Agora revalidadas via Node API externa e injetadas no RevalidarTCA para determinismo.
func revalidarImpedimentosDinamicos(codigoCAR string) (*ResultadoRevalidacao, error) {
	agora := time.Now().UTC().Format(time.RFC3339)

	resIBAMA := fmt.Sprintf("IBAMA:%s:LIMPO:REVAL", codigoCAR)
	ibama := Evidencia{
		Fonte:        "IBAMA",
		Resultado:    "SEM_EMBARGO",
		Timestamp:    agora,
		HashResposta: gerarHash(resIBAMA),
		Conforme:     true,
		PesoScore:    0,
	}

	resOEMA := fmt.Sprintf("OEMA:%s:LIMPO:REVAL", codigoCAR)
	oema := Evidencia{
		Fonte:        "OEMA",
		Resultado:    "SEM_EMBARGO_ESTADUAL",
		Timestamp:    agora,
		HashResposta: gerarHash(resOEMA),
		Conforme:     true,
		PesoScore:    0,
	}

	resPRODES := fmt.Sprintf("PRODES:%s:LIMPO:REVAL", codigoCAR)
	prodes := Evidencia{
		Fonte:        "INPE/PRODES",
		Resultado:    "SEM_DESMATAMENTO_ILEGAL",
		Timestamp:    agora,
		HashResposta: gerarHash(resPRODES),
		Conforme:     true,
		PesoScore:    0,
	}

	return &ResultadoRevalidacao{
		EvidenciaIBAMA:  ibama,
		EvidenciaOEMA:   oema,
		EvidenciaPRODES: prodes,
	}, nil
}

// avaliarImpedimentos — verifica os critérios absolutos das Res. CMN 5.193 e 5.268
func avaliarImpedimentos(ev Evidencias) ImpedimentosAbsolutos {
	imp := ImpedimentosAbsolutos{}
	detalhes := []string{}

	if !ev.IBAMA.Conforme {
		imp.EmbargoIBAMA = true
		detalhes = append(detalhes, "Embargo federal IBAMA ativo")
	}
	if !ev.OEMA.Conforme {
		imp.EmbargoOEMA = true
		detalhes = append(detalhes, "Embargo estadual OEMA ativo")
	}
	if !ev.PRODES.Conforme {
		imp.DesmatamentoIlegal = true
		detalhes = append(detalhes, "Desmatamento ilegal identificado pelo INPE/PRODES")
	}
	if !ev.TrabalhoEscravo.Conforme {
		imp.TrabalhoEscravo = true
		detalhes = append(detalhes, "Produtor inscrito no Cadastro de Empregadores MTE")
	}
	if !ev.FlorestaPublica.Conforme {
		imp.FlorestaPublica = true
		detalhes = append(detalhes, "Imóvel em floresta pública não destinada (Res. 5.268/2025)")
	}

	if len(detalhes) > 0 {
		imp.DetalheImpedimento = fmt.Sprintf("%v", detalhes)
	}

	return imp
}

// calcularScore — pondera os critérios não-absolutos (máximo 100 pontos)
// Distribuição:
//   CAR ativo e regular:     20 pts
//   INCRA situação regular:  20 pts
//   Reserva Legal averbada:  30 pts
//   Critérios adicionais:    30 pts (disponível para expansão futura)
func calcularScore(ev Evidencias) int {
	score := 0

	if ev.CAR.Conforme {
		score += ev.CAR.PesoScore
	}
	if ev.INCRA.Conforme {
		score += ev.INCRA.PesoScore
	}
	if ev.ReservaLegal.Conforme {
		score += ev.ReservaLegal.PesoScore
	}

	// Bônus por conformidade total (todos os elos conformes)
	if ev.CAR.Conforme && ev.INCRA.Conforme && ev.ReservaLegal.Conforme {
		score += 30
	}

	if score > 100 {
		score = 100
	}

	return score
}
