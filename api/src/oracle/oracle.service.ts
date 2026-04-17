import { Injectable, Logger } from '@nestjs/common';
import * as crypto from 'crypto';

@Injectable()
export class OracleService {
    private readonly logger = new Logger(OracleService.name);

    private hash(data: string): string {
        return crypto.createHash('sha256').update(data).digest('hex');
    }

    async consultarFontesAmbientais(codigoCAR: string): Promise<any> {
        this.logger.log(`Consultando fontes governamentais mockadas para CAR: ${codigoCAR}`);
        
        // Mocking behavior based on the CAR code
        const isEmbargoed = codigoCAR.includes('EMBARGADO');
        const isDesmatamento = codigoCAR.includes('DESMATADO');
        
        const timestamp = new Date().toISOString();

        return {
            car: {
                fonte: "SICAR/MMA",
                resultado: "ATIVO",
                timestamp,
                hashResposta: this.hash(`CAR:${codigoCAR}:ATIVO`),
                conforme: true,
                pesoScore: 20
            },
            ibama: {
                fonte: "IBAMA",
                resultado: isEmbargoed ? "EMBARGADO" : "SEM_EMBARGO",
                timestamp,
                hashResposta: this.hash(`IBAMA:${codigoCAR}:${isEmbargoed ? 'EMBARGADO' : 'LIMPO'}`),
                conforme: !isEmbargoed,
                pesoScore: 0
            },
            oema: {
                fonte: "OEMA",
                resultado: "SEM_EMBARGO_ESTADUAL",
                timestamp,
                hashResposta: this.hash(`OEMA:${codigoCAR}:LIMPO`),
                conforme: true,
                pesoScore: 0
            },
            incra: {
                fonte: "INCRA/SNCR",
                resultado: "SITUACAO_REGULAR",
                timestamp,
                hashResposta: this.hash(`INCRA:${codigoCAR}:REGULAR`),
                conforme: true,
                pesoScore: 20
            },
            prodes: {
                fonte: "INPE/PRODES",
                resultado: isDesmatamento ? "DESMATAMENTO_DETECTADO" : "SEM_DESMATAMENTO_ILEGAL",
                timestamp,
                hashResposta: this.hash(`PRODES:${codigoCAR}:${isDesmatamento ? 'DESMATAMENTO' : 'LIMPO'}`),
                conforme: !isDesmatamento,
                pesoScore: 0
            },
            reservaLegal: {
                fonte: "SICAR/CAR",
                resultado: "32%_AVERBADA",
                timestamp,
                hashResposta: this.hash(`RL:${codigoCAR}:32PCT_AVERBADA`),
                conforme: true,
                pesoScore: 30
            },
            trabalhoEscravo: {
                fonte: "MTE/Cadastro_Empregadores",
                resultado: "SEM_RESTRICAO",
                timestamp,
                hashResposta: this.hash(`MTE:${codigoCAR}:LIMPO`),
                conforme: true,
                pesoScore: 0
            },
            florestaPublica: {
                fonte: "SFB/Cadastro_Florestas_Publicas",
                resultado: "NAO_INCIDE_FLORESTA_PUBLICA",
                timestamp,
                hashResposta: this.hash(`FLORESTA_PUBLICA:${codigoCAR}:NAO_INCIDE`),
                conforme: true,
                pesoScore: 0
            }
        };
    }
}
