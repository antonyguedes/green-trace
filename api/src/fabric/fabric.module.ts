import { Module } from '@nestjs/common';
import { ConnectionService } from './connection.service';
import { ContractService } from './contract.service';

@Module({
  providers: [ConnectionService, ContractService],
  exports: [ContractService]
})
export class FabricModule {}
