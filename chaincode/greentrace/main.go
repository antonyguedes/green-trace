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
) (*TCA, error) {

	// 1. Verificar se já existe TCA ativo para este imóvel
	tcaExistente, err := c.buscarTCAAtivo(ctx, codigoCAR)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar TCA existente: %w", err)
	}
	if tcaExistente != nil {
		return nil, fmt.Errorf(
			"já existe TCA ativo para o imóvel %s (ID: %s). Use RevalidarTCA",
			codigoCAR, tcaExistente.ID,
		)
	}

	// 2. Identificar a IF emissora pelo MSPID do cliente
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter MSPID: %w", err)
	}

	// 3. Consultar todas as fontes e montar evidências
	evidencias, err := consultarFontesAmbientais(codigoCAR)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar fontes ambientais: %w", err)
	}

	// 4. Avaliar impedimentos absolutos (Res. CMN 5.193/2024 e 5.268/2025)
	impedimentos := avaliarImpedimentos(evidencias)

	// 5. Calcular score complementar
	score := calcularScore(evidencias)
	aprovado := !temImpedimento(impedimentos) && score >= ScoreMinimo

	// 6. Calcular datas
	agora := time.Now().UTC()
	validade := agora.AddDate(0, ValidadeMeses, 0)

	// 7. Gerar ID único
	id := fmt.Sprintf("TCA-%s-%s", codigoCAR, agora.Format("20060102150405"))

	// 8. Montar TCA
	tca := &TCA{
		ID:              id,
		CodigoCAR:       codigoCAR,
		CPFCNPJHash:     cpfCNPJHash,
		InstFinEmissora: mspID,
		Status:          StatusAtivo,
		DataEmissao:     agora.Format(time.RFC3339),
		DataValidade:    validade.Format(time.RFC3339),
		Impedimentos:    impedimentos,
		ScoreConformidade: score,
		Aprovado:        aprovado,
		Evidencias:      evidencias,
		TCAOrigemID:     "",
		RevalidadoEm:    "",
	}

	// 9. Se não aprovado, registra mesmo assim — para auditoria regulatória
	// O campo Aprovado=false sinaliza que o crédito deve ser bloqueado
	err = salvarTCA(ctx, tca)
	if err != nil {
		return nil, fmt.Errorf("erro ao salvar TCA: %w", err)
	}

	return tca, nil
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

	podeReutilizar := time.Now().UTC().Before(validade) && tca.Aprovado

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

	// 3. Revalidar APENAS impedimentos absolutos (dados dinâmicos)
	impedimentosAtuais, err := revalidarImpedimentosDinamicos(tcaOrigem.CodigoCAR)
	if err != nil {
		return nil, fmt.Errorf("erro ao revalidar impedimentos: %w", err)
	}

	// 4. Manter evidências estáticas do TCA de origem (CAR, INCRA, RL)
	evidencias := tcaOrigem.Evidencias
	evidencias.IBAMA = impedimentosAtuais.EvidenciaIBAMA
	evidencias.OEMA = impedimentosAtuais.EvidenciaOEMA
	evidencias.PRODES = impedimentosAtuais.EvidenciaPRODES

	// 5. Recalcular
	impedimentos := avaliarImpedimentos(evidencias)
	score := calcularScore(evidencias)
	aprovado := !temImpedimento(impedimentos) && score >= ScoreMinimo

	// 6. Novo TCA vinculado à IF solicitante
	agora := time.Now().UTC()
	validade := agora.AddDate(0, ValidadeMeses, 0)
	id := fmt.Sprintf("TCA-%s-%s-REVAL", tcaOrigem.CodigoCAR, agora.Format("20060102150405"))

	novoTCA := &TCA{
		ID:                id,
		CodigoCAR:         tcaOrigem.CodigoCAR,
		CPFCNPJHash:       cpfCNPJHash,
		InstFinEmissora:   mspID,
		Status:            StatusAtivo,
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
	agora := time.Now().UTC()
	tca.Status = StatusRevogado
	tca.DataRevogacao = agora.Format(time.RFC3339)
	tca.MotivoRevogacao = motivo

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

	agora := time.Now().UTC()
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
