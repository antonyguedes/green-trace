import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { DocumentBuilder, SwaggerModule } from '@nestjs/swagger';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  
  // CORS Configuration
  app.enableCors({
    origin: '*', // Permite frontend (Vite)
    methods: 'GET,HEAD,PUT,PATCH,POST,DELETE',
    preflightContinue: false,
    optionsSuccessStatus: 204,
  });

  // Swagger Configuration
  const config = new DocumentBuilder()
    .setTitle('GreenTrace API')
    .setDescription('API Gateway REST para comunicação com o Ledger do sistema de Conformidade Ambiental (TCA)')
    .setVersion('1.0')
    .addTag('tca')
    .build();
    
  const document = SwaggerModule.createDocument(app, config);
  SwaggerModule.setup('api/docs', app, document);

  await app.listen(3000);
  console.log(`Application is running on: http://localhost:3000`);
  console.log(`Swagger docs at: http://localhost:3000/api/docs`);
}
bootstrap();
