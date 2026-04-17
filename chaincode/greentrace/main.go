package main

// ─────────────────────────────────────────────────────────────────────────────
// main.go — Smart contract GreenTrace
// Funções: EmitirTCA | ConsultarTCA | RevogarTCA | ListarTCAs | RevalidarTCA
// ─────────────────────────────────────────────────────────────────────────────

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// GreenTraceContract — contrato principal
type GreenTraceContract struct {
	contractapi.Contract
}

// ─────────────────────────────────────────────────────────────────────────────
// EmitirTCA
// Consulta as fontes ambientais, avalia impedimentos e score,
// e registra o TCA no ledger se aprovado.
// Chamado pela IF ao iniciar análise de crédito rural.
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) EmitirTCA(
    ctx contractapi.TransactionContextInterface,
    codigoCAR string,
    cpfCNPJHash string,
    evidenciasJSON string, // Recebido do Backend (NestJS)
) (*TCA, error) {

    // 1. DETERMINISMO: Obter tempo da transação
    txTimestamp, err := ctx.GetStub().GetTxTimestamp()
    if err != nil {
        return nil, fmt.Errorf("erro timestamp: %w", err)
    }
    agora := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC()

    // 2. BUSCA EXISTENTE (Usando sua função auxiliar)
    tcaExistente, _ := c.buscarTCAAtivo(ctx, codigoCAR)
    if tcaExistente != nil {
        return nil, fmt.Errorf("já existe TCA ativo para o imóvel %s", codigoCAR)
    }

    // 3. PARSE DAS EVIDÊNCIAS
    var ev Evidencias
    if err := json.Unmarshal([]byte(evidenciasJSON), &ev); err != nil {
        return nil, fmt.Errorf("falha ao processar evidências: %v", err)
    }

    // 4. LÓGICA DE NEGÓCIO (Baseada no seu Model)
    impedimentos := avaliarImpedimentos(ev)

    // Cálculo de Score simples baseado nos pesos que você definiu no Model
    scoreTotal := ev.CAR.PesoScore + ev.IBAMA.PesoScore + ev.PRODES.PesoScore // ... etc
    aprovado := !temImpedimento(impedimentos) && scoreTotal >= ScoreMinimo

    // 5. MONTAGEM DO OBJETO
    status := StatusAtivo
    if !aprovado {
        status = StatusNegado
    }

    mspID, _ := ctx.GetClientIdentity().GetMSPID()
    tca := &TCA{
        ID:                fmt.Sprintf("TCA-%s-%d", codigoCAR, txTimestamp.Seconds),
        CodigoCAR:         codigoCAR,
        CPFCNPJHash:       cpfCNPJHash,
        InstFinEmissora:   mspID,
        Status:            status,
        DataEmissao:       agora.Format(time.RFC3339),
        DataValidade:      agora.AddDate(0, ValidadeMeses, 0).Format(time.RFC3339),
        Impedimentos:      impedimentos,
        ScoreConformidade: scoreTotal,
        Aprovado:          aprovado,
        Evidencias:        ev,
    }

    // 6. PERSISTÊNCIA
    return tca, salvarTCA(ctx, tca)
}

// ─────────────────────────────────────────────────────────────────────────────
// ConsultarTCA
// Busca o TCA mais recente para um imóvel.
// Qualquer IF do canal pode consultar — sem expor dados brutos.
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) ConsultarTCA(
	ctx contractapi.TransactionContextInterface,
	codigoCAR string,
) (*RespostaConsulta, error) {

	tca, err := c.buscarTCAAtivo(ctx, codigoCAR)
	if err != nil {
		return nil, err
	}

	if tca == nil {
		return &RespostaConsulta{
			Encontrado:    false,
			Mensagem:      "Nenhum TCA ativo encontrado para este imóvel",
			PodeReutilizar: false,
		}, nil
	}

	// Verificar se ainda está dentro da validade
	validade, err := time.Parse(time.RFC3339, tca.DataValidade)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear data de validade: %w", err)
	}
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter timestamp da transação: %v", err)
	}
	agora := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC()

	podeReutilizar := agora.Before(validade) && tca.Aprovado

	return &RespostaConsulta{
		Encontrado:    true,
		TCA:           tca,
		PodeReutilizar: podeReutilizar,
		Mensagem: func() string {
			if podeReutilizar {
				return fmt.Sprintf("TCA válido até %s. Use RevalidarTCA para emitir novo token vinculado à sua IF.", tca.DataValidade)
			}
			return "TCA encontrado mas expirado ou não aprovado. Emita novo TCA."
		}(),
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RevalidarTCA
// Quando uma segunda IF quer usar um imóvel já verificado.
// Revalida impedimentos absolutos (IBAMA + PRODES mudam frequentemente)
// e reaproveitam evidências estáticas (CAR, INCRA, RL).
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) RevalidarTCA(
	ctx contractapi.TransactionContextInterface,
	tcaOrigemID string,
	cpfCNPJHash string,
	evidenciasJSON string,
) (*TCA, error) {

	// 1. Buscar TCA de origem
	tcaOrigem, err := buscarTCAPorID(ctx, tcaOrigemID)
	if err != nil {
		return nil, fmt.Errorf("TCA de origem não encontrado: %w", err)
	}

	if tcaOrigem.Status != StatusAtivo {
		return nil, fmt.Errorf("TCA de origem está %s — não pode ser revalidado", tcaOrigem.Status)
	}

	// 2. MSPID da nova IF
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter MSPID: %w", err)
	}

	// 3. Processar novas evidências dinâmicas vindas do cliente
	var novasEvidencias Evidencias
	if err := json.Unmarshal([]byte(evidenciasJSON), &novasEvidencias); err != nil {
		return nil, fmt.Errorf("falha ao processar evidências para revalidação: %v", err)
	}

	// 4. Manter evidências estáticas do TCA de origem (CAR, INCRA, RL)
	evidencias := tcaOrigem.Evidencias
	evidencias.IBAMA = novasEvidencias.IBAMA
	evidencias.OEMA = novasEvidencias.OEMA
	evidencias.PRODES = novasEvidencias.PRODES
	evidencias.TrabalhoEscravo = novasEvidencias.TrabalhoEscravo
	evidencias.FlorestaPublica = novasEvidencias.FlorestaPublica

	// 5. Recalcular
	impedimentos := avaliarImpedimentos(evidencias)
	score := calcularScore(evidencias)
	aprovado := !temImpedimento(impedimentos) && score >= ScoreMinimo

	status := StatusAtivo
	if !aprovado {
		status = StatusNegado
	}

	// 6. Novo TCA vinculado à IF solicitante
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter timestamp da transação: %v", err)
	}
	agora := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC()
	validade := agora.AddDate(0, ValidadeMeses, 0)
	id := fmt.Sprintf("TCA-%s-%s-REVAL", tcaOrigem.CodigoCAR, agora.Format("20060102150405"))

	novoTCA := &TCA{
		ID:                id,
		CodigoCAR:         tcaOrigem.CodigoCAR,
		CPFCNPJHash:       cpfCNPJHash,
		InstFinEmissora:   mspID,
		Status:            status,
		DataEmissao:       agora.Format(time.RFC3339),
		DataValidade:      validade.Format(time.RFC3339),
		Impedimentos:      impedimentos,
		ScoreConformidade: score,
		Aprovado:          aprovado,
		Evidencias:        evidencias,
		TCAOrigemID:       tcaOrigemID,
		RevalidadoEm:      agora.Format(time.RFC3339),
	}

	err = salvarTCA(ctx, novoTCA)
	if err != nil {
		return nil, fmt.Errorf("erro ao salvar TCA revalidado: %w", err)
	}

	return novoTCA, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RevogarTCA
// Revoga um TCA ativo — usado quando surge novo embargo ou
// informação superveniente de irregularidade.
// Restrito ao BC (BancoCentralMSP) ou à IF emissora.
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) RevogarTCA(
	ctx contractapi.TransactionContextInterface,
	tcaID string,
	motivo string,
) error {

	// 1. Controle de acesso — só BC ou IF emissora pode revogar
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("erro ao obter MSPID: %w", err)
	}

	tca, err := buscarTCAPorID(ctx, tcaID)
	if err != nil {
		return fmt.Errorf("TCA não encontrado: %w", err)
	}

	if mspID != "BancoCentralMSP" && mspID != tca.InstFinEmissora {
		return fmt.Errorf("acesso negado: apenas BancoCentralMSP ou a IF emissora podem revogar")
	}

	if tca.Status == StatusRevogado {
		return fmt.Errorf("TCA já está revogado")
	}

	// 2. Atualizar status
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("falha ao obter timestamp da transação: %v", err)
	}
	agora := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC()
	tca.Status = StatusRevogado
	tca.DataRevogacao = agora.Format(time.RFC3339)
	tca.MotivoRevogacao = motivo

	return salvarTCA(ctx, tca)
}

// ─────────────────────────────────────────────────────────────────────────────
// SuspenderTCA
// Suspende um TCA ATIVO. Geralmente motivado por alertas pós-emissão.
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) SuspenderTCA(
	ctx contractapi.TransactionContextInterface,
	tcaID string,
	motivo string,
) error {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("erro ao obter MSPID: %w", err)
	}

	tca, err := buscarTCAPorID(ctx, tcaID)
	if err != nil {
		return fmt.Errorf("TCA não encontrado: %w", err)
	}

	if mspID != "BancoCentralMSP" && mspID != tca.InstFinEmissora {
		return fmt.Errorf("acesso negado: apenas BancoCentralMSP ou a IF emissora podem suspender")
	}

	if tca.Status != StatusAtivo {
		return fmt.Errorf("não é possível suspender um TCA com status %s", tca.Status)
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("falha ao obter timestamp: %v", err)
	}
	agora := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC().Format(time.RFC3339)

	tca.Status = StatusSuspenso
	tca.DataSuspensao = agora
	tca.MotivoSuspensao = motivo

	tca.HistoricoSuspensoes = append(tca.HistoricoSuspensoes, Suspensao{
		Motivo:       motivo,
		DataSuspenso: agora,
	})

	return salvarTCA(ctx, tca)
}

// ─────────────────────────────────────────────────────────────────────────────
// ReativarTCA
// Retorna um TCA SUSPENSO para ATIVO após validações adicionais
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) ReativarTCA(
	ctx contractapi.TransactionContextInterface,
	tcaID string,
	evidenciasJSON string,
) (*TCA, error) {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter MSPID: %w", err)
	}

	tca, err := buscarTCAPorID(ctx, tcaID)
	if err != nil {
		return nil, fmt.Errorf("TCA não encontrado: %w", err)
	}

	if mspID != "BancoCentralMSP" && mspID != tca.InstFinEmissora {
		return nil, fmt.Errorf("acesso negado: apenas BancoCentralMSP ou a IF emissora podem reativar")
	}

	if tca.Status != StatusSuspenso {
		return nil, fmt.Errorf("apenas TCAs suspensos podem ser reativados")
	}

	var novasEvidencias Evidencias
	if err := json.Unmarshal([]byte(evidenciasJSON), &novasEvidencias); err != nil {
		return nil, fmt.Errorf("falha ao processar evidências: %v", err)
	}

	impedimentos := avaliarImpedimentos(novasEvidencias)
	if temImpedimento(impedimentos) {
		return nil, fmt.Errorf("evidências fornecidas ainda constam impedimentos, não é possível reativar")
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter timestamp: %v", err)
	}
	agora := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC()

	// Marca a última suspensão como reativada
	if len(tca.HistoricoSuspensoes) > 0 {
		tca.HistoricoSuspensoes[len(tca.HistoricoSuspensoes)-1].ReativadoEm = agora.Format(time.RFC3339)
	}

	tca.Status = StatusAtivo
	tca.MotivoSuspensao = ""
	tca.Evidencias = novasEvidencias
	tca.Impedimentos = impedimentos
	tca.ScoreConformidade = calcularScore(novasEvidencias)
	tca.Aprovado = true

	// Se o TCA for vencer nos próximos 30 dias (ou já estourou na janela de suspensão), estender validade
	validadeAtual, err := time.Parse(time.RFC3339, tca.DataValidade)
	if err == nil && validadeAtual.Before(agora.AddDate(0, 1, 0)) {
		tca.DataValidade = agora.AddDate(0, ValidadeMeses, 0).Format(time.RFC3339)
	}

	if err := salvarTCA(ctx, tca); err != nil {
		return nil, err
	}

	return tca, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FinalizarTCA
// Marca o TCA como FINALIZADO caso a linha de crédito acabe ou seja quitada.
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) FinalizarTCA(
	ctx contractapi.TransactionContextInterface,
	tcaID string,
) error {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("erro ao obter MSPID: %w", err)
	}

	tca, err := buscarTCAPorID(ctx, tcaID)
	if err != nil {
		return fmt.Errorf("TCA não encontrado: %w", err)
	}

	if mspID != tca.InstFinEmissora && mspID != "BancoCentralMSP" {
		return fmt.Errorf("apenas a IF emissora ou o Bacen podem finalizar este TCA")
	}

	if tca.Status == StatusRevogado || tca.Status == StatusFinalizado {
		return fmt.Errorf("TCA com status irrevogável (%s) não pode ser finalizado", tca.Status)
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("falha ao obter timestamp: %v", err)
	}
	agora := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC().Format(time.RFC3339)

	tca.Status = StatusFinalizado
	tca.DataFinalizacao = agora

	return salvarTCA(ctx, tca)
}

// ─────────────────────────────────────────────────────────────────────────────
// ListarTCAs
// Retorna todos os TCAs do ledger — visão regulatória para o BC.
// Restrito ao BancoCentralMSP.
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) ListarTCAs(
	ctx contractapi.TransactionContextInterface,
) ([]*TCA, error) {

	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter MSPID: %w", err)
	}

	if mspID != "BancoCentralMSP" {
		return nil, fmt.Errorf("acesso negado: apenas BancoCentralMSP pode listar todos os TCAs")
	}

	// Rich query — requer CouchDB como state DB
	query := `{"selector": {"_id": {"$gt": "TCA-"}}}`
	iterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return nil, fmt.Errorf("erro na query: %w", err)
	}
	defer iterator.Close()

	var tcas []*TCA
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		var tca TCA
		if err := json.Unmarshal(item.Value, &tca); err != nil {
			return nil, err
		}
		tcas = append(tcas, &tca)
	}

	return tcas, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ConsultarMeusTCAs
// Retorna os TCAs onde a IF logada é a emissora.
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) ConsultarMeusTCAs(
	ctx contractapi.TransactionContextInterface,
) ([]*TCA, error) {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter MSPID: %w", err)
	}

	query := fmt.Sprintf(`{"selector": {"instFinEmissora": "%s"}}`, mspID)
	iterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return nil, fmt.Errorf("erro na query: %w", err)
	}
	defer iterator.Close()

	var tcas []*TCA
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		var tca TCA
		if err := json.Unmarshal(item.Value, &tca); err != nil {
			return nil, err
		}
		tcas = append(tcas, &tca)
	}

	return tcas, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// AuditarTransacoes
// Retorna o histórico de estado de um TCA baseado no Código CAR
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) AuditarTransacoes(
	ctx contractapi.TransactionContextInterface,
	codigoCAR string,
) ([]*HistoricoTCA, error) {
	// Pega todos os TCAs emitidos para esse CAR (pode haver mais de um, por ex originais e revalidações)
	query := fmt.Sprintf(`{"selector": {"codigoCAR": "%s"}}`, codigoCAR)
	iterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return nil, fmt.Errorf("erro na query: %w", err)
	}
	defer iterator.Close()

	var resultados []*HistoricoTCA
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		
		// Pra cada TCA, vamos buscar as transações daquele UUID (Key)
		var tca TCA
		if err := json.Unmarshal(item.Value, &tca); err != nil {
			continue
		}

		histIterator, err := ctx.GetStub().GetHistoryForKey(tca.ID)
		if err != nil {
			return nil, fmt.Errorf("erro lendo history para a key %s: %v", tca.ID, err)
		}

		for histIterator.HasNext() {
			histRecord, err := histIterator.Next()
			if err != nil {
				return nil, err
			}
			var histTCA TCA
			if err := json.Unmarshal(histRecord.Value, &histTCA); err == nil {
				ts := time.Unix(histRecord.Timestamp.Seconds, int64(histRecord.Timestamp.Nanos)).UTC().Format(time.RFC3339)
				resultados = append(resultados, &HistoricoTCA{
					TxId:      histRecord.TxId,
					Timestamp: ts,
					TCA:       &histTCA,
				})
			}
		}
		histIterator.Close()
	}

	return resultados, nil
}


// ─────────────────────────────────────────────────────────────────────────────
// ExpirarTCAs
// Marca TCAs vencidos como EXPIRADO.
// Chamado periodicamente (ex: via scheduler externo ou trigger).
// ─────────────────────────────────────────────────────────────────────────────
func (c *GreenTraceContract) ExpirarTCAs(
	ctx contractapi.TransactionContextInterface,
) (int, error) {

	query := `{"selector": {"status": "ATIVO"}}`
	iterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return 0, fmt.Errorf("erro na query: %w", err)
	}
	defer iterator.Close()

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return 0, fmt.Errorf("falha ao obter timestamp da transação: %v", err)
	}
	agora := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC()
	count := 0

	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return count, err
		}
		var tca TCA
		if err := json.Unmarshal(item.Value, &tca); err != nil {
			return count, err
		}

		validade, err := time.Parse(time.RFC3339, tca.DataValidade)
		if err != nil {
			continue
		}

		if agora.After(validade) {
			tca.Status = StatusExpirado
			if err := salvarTCA(ctx, &tca); err != nil {
				return count, err
			}
			count++
		}
	}

	return count, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Funções auxiliares internas
// ─────────────────────────────────────────────────────────────────────────────

func salvarTCA(ctx contractapi.TransactionContextInterface, tca *TCA) error {
	bytes, err := json.Marshal(tca)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(tca.ID, bytes)
}

func buscarTCAPorID(ctx contractapi.TransactionContextInterface, id string) (*TCA, error) {
	bytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, err
	}
	if bytes == nil {
		return nil, fmt.Errorf("TCA %s não encontrado", id)
	}
	var tca TCA
	if err := json.Unmarshal(bytes, &tca); err != nil {
		return nil, err
	}
	return &tca, nil
}

func (c *GreenTraceContract) buscarTCAAtivo(
	ctx contractapi.TransactionContextInterface,
	codigoCAR string,
) (*TCA, error) {
	query := fmt.Sprintf(`{"selector": {"codigoCAR": "%s", "status": "ATIVO"}}`, codigoCAR)
	iterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	if iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, err
		}
		var tca TCA
		if err := json.Unmarshal(item.Value, &tca); err != nil {
			return nil, err
		}
		return &tca, nil
	}
	return nil, nil
}

func temImpedimento(imp ImpedimentosAbsolutos) bool {
	return imp.EmbargoIBAMA ||
		imp.EmbargoOEMA ||
		imp.DesmatamentoIlegal ||
		imp.TrabalhoEscravo ||
		imp.FlorestaPublica
}

// gerarHash — gera SHA-256 de uma string (usado para hashear CPF/CNPJ e respostas de API)
func gerarHash(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ─────────────────────────────────────────────────────────────────────────────
// main
// ─────────────────────────────────────────────────────────────────────────────
func main() {
	chaincode, err := contractapi.NewChaincode(&GreenTraceContract{})
	if err != nil {
		panic(fmt.Sprintf("erro ao criar chaincode: %v", err))
	}
	if err := chaincode.Start(); err != nil {
		panic(fmt.Sprintf("erro ao iniciar chaincode: %v", err))
	}
}
