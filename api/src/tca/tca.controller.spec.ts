import { Test, TestingModule } from '@nestjs/testing';
import { TcaController } from './tca.controller';

describe('TcaController', () => {
  let controller: TcaController;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [TcaController],
    }).compile();

    controller = module.get<TcaController>(TcaController);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });
});
