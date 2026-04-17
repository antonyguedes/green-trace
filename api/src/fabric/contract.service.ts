import { Injectable, Logger } from '@nestjs/common';
import { ConnectionService } from './connection.service';

@Injectable()
export class ContractService {
    private readonly logger = new Logger(ContractService.name);

    constructor(private readonly connectionService: ConnectionService) {}

    private utf8Decoder = new TextDecoder();

    async emitirTCA(orgName: string, codigoCAR: string, cpfCNPJHash: string, evidenciasJSON: string): Promise<any> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Emitindo TCA para CAR: ${codigoCAR}`);
        
        const resultBytes = await contract.submitTransaction('EmitirTCA', codigoCAR, cpfCNPJHash, evidenciasJSON);
        const resultJson = this.utf8Decoder.decode(resultBytes);
        return JSON.parse(resultJson);
    }

    async consultarTCA(orgName: string, codigoCAR: string): Promise<any> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Consultando TCA para CAR: ${codigoCAR}`);
        
        const resultBytes = await contract.evaluateTransaction('ConsultarTCA', codigoCAR);
        const resultJson = this.utf8Decoder.decode(resultBytes);
        return JSON.parse(resultJson);
    }

    async revalidarTCA(orgName: string, tcaOrigemID: string, cpfCNPJHash: string, evidenciasJSON: string): Promise<any> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Revalidando TCA ID: ${tcaOrigemID}`);
        
        const resultBytes = await contract.submitTransaction('RevalidarTCA', tcaOrigemID, cpfCNPJHash, evidenciasJSON);
        const resultJson = this.utf8Decoder.decode(resultBytes);
        return JSON.parse(resultJson);
    }

    async suspenderTCA(orgName: string, tcaID: string, motivo: string): Promise<void> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Suspendendo TCA ID: ${tcaID} (Motivo: ${motivo})`);
        await contract.submitTransaction('SuspenderTCA', tcaID, motivo);
    }

    async reativarTCA(orgName: string, tcaID: string, evidenciasJSON: string): Promise<any> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Reativando TCA ID: ${tcaID}`);
        
        const resultBytes = await contract.submitTransaction('ReativarTCA', tcaID, evidenciasJSON);
        const resultJson = this.utf8Decoder.decode(resultBytes);
        return JSON.parse(resultJson);
    }

    async finalizarTCA(orgName: string, tcaID: string): Promise<void> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Finalizando TCA ID: ${tcaID}`);
        await contract.submitTransaction('FinalizarTCA', tcaID);
    }

    async revogarTCA(orgName: string, tcaID: string, motivo: string): Promise<void> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Revogando TCA ID: ${tcaID} (Motivo: ${motivo})`);
        await contract.submitTransaction('RevogarTCA', tcaID, motivo);
    }

    async listarTCAs(orgName: string): Promise<any[]> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Listando todos os TCAs do canal`);
        
        let resultBytes;
        try {
            resultBytes = await contract.evaluateTransaction('ListarTCAs');
        } catch (e) { return [] }
        if (!resultBytes || resultBytes.length === 0) return [];
        const resultJson = this.utf8Decoder.decode(resultBytes);
        return JSON.parse(resultJson);
    }

    async consultarMeusTCAs(orgName: string): Promise<any[]> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Consultando TCAs emitidos por esta instituição`);
        
        let resultBytes;
        try {
            resultBytes = await contract.evaluateTransaction('ConsultarMeusTCAs');
        } catch (e) { return [] }
        if (!resultBytes || resultBytes.length === 0) return [];
        const resultJson = this.utf8Decoder.decode(resultBytes);
        return JSON.parse(resultJson);
    }

    async auditarTransacoes(orgName: string, codigoCAR: string): Promise<any[]> {
        const contract = await this.connectionService.getContract(orgName);
        this.logger.log(`[${orgName}] Auditando transações do CAR: ${codigoCAR}`);
        
        let resultBytes;
        try {
            resultBytes = await contract.evaluateTransaction('AuditarTransacoes', codigoCAR);
        } catch (e) { return [] }
        if (!resultBytes || resultBytes.length === 0) return [];
        const resultJson = this.utf8Decoder.decode(resultBytes);
        return JSON.parse(resultJson);
    }
}
