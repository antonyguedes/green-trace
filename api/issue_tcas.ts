import { NestFactory } from '@nestjs/core';
import { AppModule } from './src/app.module';
import { TcaController } from './src/tca/tca.controller';

async function emitir() {
  const app = await NestFactory.createApplicationContext(AppModule);
  const controller = app.get(TcaController);
  
  console.log("Emitindo TCA 1...");
  await controller.emitirTCA('InstFinA', { codigoCAR: 'SP-1234567-0001', cpfCNPJHash: 'hash1' });
  
  console.log("Emitindo TCA 2...");
  await controller.emitirTCA('InstFinA', { codigoCAR: 'MG-7654321-0002', cpfCNPJHash: 'hash2' });

  console.log("Emitindo TCA 3 (Embargado)...");
  await controller.emitirTCA('InstFinB', { codigoCAR: 'MT-EMBARGADO-0003', cpfCNPJHash: 'hash3' });

  console.log("Emitindo TCA 4 (Desmatamento)...");
  await controller.emitirTCA('InstFinB', { codigoCAR: 'PA-DESMATADO-0004', cpfCNPJHash: 'hash4' });

  console.log("Concluído!");
  process.exit(0);
}

emitir().catch(err => {
  console.error(err);
  process.exit(1);
});
