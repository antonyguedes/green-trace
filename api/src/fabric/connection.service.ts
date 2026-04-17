import { Injectable, Logger } from '@nestjs/common';
import * as grpc from '@grpc/grpc-js';
import { connect, Contract, Gateway, Identity, Signer, signers } from '@hyperledger/fabric-gateway';
import * as crypto from 'crypto';
import * as fs from 'fs/promises';
import * as path from 'path';

export interface OrgConfig {
    mspId: string;
    cryptoPath: string;
    tlsCertPath: string;
    peerEndpoint: string;
    peerHostAlias: string;
}

@Injectable()
export class ConnectionService {
    private readonly logger = new Logger(ConnectionService.name);
    private gateways: Map<string, Gateway> = new Map();
    private readonly channelName = 'greentracechannel';
    private readonly chaincodeName = 'greentrace';

    // Para o ambiente de dev (docker-compose local)
    private readonly orgConfigs: Record<string, OrgConfig> = {
        BancoCentral: {
            mspId: 'BancoCentralMSP',
            cryptoPath: path.resolve(__dirname, '../../../../network/crypto-config/peerOrganizations/bancocentral.green-trace.com'),
            tlsCertPath: path.resolve(__dirname, '../../../../network/crypto-config/peerOrganizations/bancocentral.green-trace.com/peers/peer0.bancocentral.green-trace.com/tls/ca.crt'),
            peerEndpoint: 'localhost:7051',
            peerHostAlias: 'peer0.bancocentral.green-trace.com',
        },
        InstFinA: {
            mspId: 'InstFinAMSP',
            cryptoPath: path.resolve(__dirname, '../../../../network/crypto-config/peerOrganizations/instfina.green-trace.com'),
            tlsCertPath: path.resolve(__dirname, '../../../../network/crypto-config/peerOrganizations/instfina.green-trace.com/peers/peer0.instfina.green-trace.com/tls/ca.crt'),
            peerEndpoint: 'localhost:8051',
            peerHostAlias: 'peer0.instfina.green-trace.com',
        },
        InstFinB: {
            mspId: 'InstFinBMSP',
            cryptoPath: path.resolve(__dirname, '../../../../network/crypto-config/peerOrganizations/instfinb.green-trace.com'),
            tlsCertPath: path.resolve(__dirname, '../../../../network/crypto-config/peerOrganizations/instfinb.green-trace.com/peers/peer0.instfinb.green-trace.com/tls/ca.crt'),
            peerEndpoint: 'localhost:9051',
            peerHostAlias: 'peer0.instfinb.green-trace.com',
        },
        OrgAmbiental: {
            mspId: 'OrgAmbientalMSP',
            cryptoPath: path.resolve(__dirname, '../../../../network/crypto-config/peerOrganizations/orgambiental.green-trace.com'),
            tlsCertPath: path.resolve(__dirname, '../../../../network/crypto-config/peerOrganizations/orgambiental.green-trace.com/peers/peer0.orgambiental.green-trace.com/tls/ca.crt'),
            peerEndpoint: 'localhost:10051',
            peerHostAlias: 'peer0.orgambiental.green-trace.com',
        }
    };

    async getContract(orgName: string): Promise<Contract> {
        let gateway = this.gateways.get(orgName);
        if (!gateway) {
            this.logger.log(`Criando nova conexão Gateway para a Organização: ${orgName}...`);
            const config = this.orgConfigs[orgName];
            if (!config) throw new Error(`Organização desconhecida: ${orgName}`);

            const client = await this.newGrpcConnection(config);
            const gatewayConn = connect({
                client,
                identity: await this.newIdentity(config),
                signer: await this.newSigner(config),
                evaluateOptions: () => ({ deadline: Date.now() + 5000 }),
                endorseOptions: () => ({ deadline: Date.now() + 15000 }),
                submitOptions: () => ({ deadline: Date.now() + 5000 }),
                commitStatusOptions: () => ({ deadline: Date.now() + 60000 }),
            });

            this.gateways.set(orgName, gatewayConn);
            gateway = gatewayConn;
        }

        const network = gateway.getNetwork(this.channelName);
        return network.getContract(this.chaincodeName);
    }

    private async newGrpcConnection(config: OrgConfig): Promise<grpc.Client> {
        const tlsRootCert = await fs.readFile(config.tlsCertPath);
        const tlsCredentials = grpc.credentials.createSsl(tlsRootCert);
        return new grpc.Client(config.peerEndpoint, tlsCredentials, {
            'grpc.ssl_target_name_override': config.peerHostAlias,
            'grpc.max_receive_message_length': 10 * 1024 * 1024,
        });
    }

    private async newIdentity(config: OrgConfig): Promise<Identity> {
        const domain = config.peerHostAlias.replace('peer0.', '');
        const certPath = path.join(config.cryptoPath, `users/Admin@${domain}/msp/signcerts/Admin@${domain}-cert.pem`);
        const credentials = await fs.readFile(certPath);
        return { mspId: config.mspId, credentials };
    }

    private async newSigner(config: OrgConfig): Promise<Signer> {
        const domain = config.peerHostAlias.replace('peer0.', '');
        const keyDirPath = path.join(config.cryptoPath, `users/Admin@${domain}/msp/keystore`);
        const files = await fs.readdir(keyDirPath);
        const keyPath = path.join(keyDirPath, files[0]);
        const privateKeyPem = await fs.readFile(keyPath);
        const privateKey = crypto.createPrivateKey(privateKeyPem);
        return signers.newPrivateKeySigner(privateKey);
    }
}
