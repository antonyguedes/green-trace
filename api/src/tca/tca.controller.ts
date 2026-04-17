import { Controller, Post, Get, Body, Param, Headers, HttpException, HttpStatus } from '@nestjs/common';
import { ContractService } from '../fabric/contract.service';
import { OracleService } from '../oracle/oracle.service';
import { ApiTags, ApiHeader, ApiOperation } from '@nestjs/swagger';

@ApiTags('tca')
@ApiHeader({
  name: 'x-organization',
  description: 'Mock identity (BancoCentral, InstFinA, InstFinB, OrgAmbiental)',
  required: true,
})
@Controller('tca')
export class TcaController {
    constructor(
        private readonly contractService: ContractService,
        private readonly oracleService: OracleService
    ) {}

    private getOrg(orgName?: string): string {
        if (!orgName) {
            throw new HttpException('Missing x-organization header', HttpStatus.UNAUTHORIZED);
        }
        return orgName;
    }

    @Post()
    @ApiOperation({ summary: 'Emitir um novo TCA' })
    async emitirTCA(
        @Headers('x-organization') org: string,
        @Body() body: { codigoCAR: string, cpfCNPJHash: string }
    ) {
        const orgName = this.getOrg(org);
        const evidencias = await this.oracleService.consultarFontesAmbientais(body.codigoCAR);
        return await this.contractService.emitirTCA(orgName, body.codigoCAR, body.cpfCNPJHash, JSON.stringify(evidencias));
    }

    @Get('meus')
    @ApiOperation({ summary: 'Listar TCAs da Instituição Financeira Atual' })
    async consultarMeusTCAs(@Headers('x-organization') org: string) {
        const orgName = this.getOrg(org);
        return await this.contractService.consultarMeusTCAs(orgName);
    }

    @Get('todos')
    @ApiOperation({ summary: 'Banco Central - Listar todos os TCAs do ledger' })
    async listarTCAs(@Headers('x-organization') org: string) {
        const orgName = this.getOrg(org);
        return await this.contractService.listarTCAs(orgName);
    }

    @Get(':codigoCAR/historico')
    @ApiOperation({ summary: 'Banco Central - Auditar transações do CAR' })
    async auditarTransacoes(@Headers('x-organization') org: string, @Param('codigoCAR') codigoCAR: string) {
        const orgName = this.getOrg(org);
        return await this.contractService.auditarTransacoes(orgName, codigoCAR);
    }

    @Get(':codigoCAR')
    @ApiOperation({ summary: 'Consultar estado atual do TCA' })
    async consultarTCA(@Headers('x-organization') org: string, @Param('codigoCAR') codigoCAR: string) {
        const orgName = this.getOrg(org);
        return await this.contractService.consultarTCA(orgName, codigoCAR);
    }

    @Post('revalidar')
    @ApiOperation({ summary: 'Revalidar um TCA emitido por outra IF' })
    async revalidarTCA(
        @Headers('x-organization') org: string,
        @Body() body: { tcaOrigemID: string, cpfCNPJHash: string, codigoCAR: string }
    ) {
        const orgName = this.getOrg(org);
        const evidencias = await this.oracleService.consultarFontesAmbientais(body.codigoCAR);
        return await this.contractService.revalidarTCA(orgName, body.tcaOrigemID, body.cpfCNPJHash, JSON.stringify(evidencias));
    }

    @Post(':id/suspender')
    @ApiOperation({ summary: 'Suspender um TCA' })
    async suspenderTCA(
        @Headers('x-organization') org: string,
        @Param('id') id: string,
        @Body() body: { motivo: string }
    ) {
        const orgName = this.getOrg(org);
        await this.contractService.suspenderTCA(orgName, id, body.motivo);
        return { success: true };
    }

    @Post(':id/reativar')
    @ApiOperation({ summary: 'Reativar um TCA Suspenso' })
    async reativarTCA(
        @Headers('x-organization') org: string,
        @Param('id') id: string,
        @Body() body: { codigoCAR: string }
    ) {
        const orgName = this.getOrg(org);
        const evidencias = await this.oracleService.consultarFontesAmbientais(body.codigoCAR);
        return await this.contractService.reativarTCA(orgName, id, JSON.stringify(evidencias));
    }

    @Post(':id/finalizar')
    @ApiOperation({ summary: 'Finalizar um TCA' })
    async finalizarTCA(
        @Headers('x-organization') org: string,
        @Param('id') id: string
    ) {
        const orgName = this.getOrg(org);
        await this.contractService.finalizarTCA(orgName, id);
        return { success: true };
    }

    @Post(':id/revogar')
    @ApiOperation({ summary: 'Revogar um TCA' })
    async revogarTCA(
        @Headers('x-organization') org: string,
        @Param('id') id: string,
        @Body() body: { motivo: string }
    ) {
        const orgName = this.getOrg(org);
        await this.contractService.revogarTCA(orgName, id, body.motivo);
        return { success: true };
    }
}
