import { Module } from '@nestjs/common';
import { TcaController } from './tca.controller';
import { FabricModule } from '../fabric/fabric.module';
import { OracleModule } from '../oracle/oracle.module';

@Module({
  imports: [FabricModule, OracleModule],
  controllers: [TcaController]
})
export class TcaModule {}
