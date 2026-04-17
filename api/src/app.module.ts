import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
import { FabricModule } from './fabric/fabric.module';
import { OracleModule } from './oracle/oracle.module';
import { TcaModule } from './tca/tca.module';

@Module({
  imports: [FabricModule, OracleModule, TcaModule],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
