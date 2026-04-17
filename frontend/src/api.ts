import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:3000',
});

// Interceptor para injeção do Org (simulação Auth)
api.interceptors.request.use((config) => {
  const org = localStorage.getItem('x-organization') || 'BancoCentral';
  config.headers['x-organization'] = org;
  return config;
});

export const tcaApi = {
  setOrg: (org: string) => {
    localStorage.setItem('x-organization', org);
  },
  getOrg: () => {
    return localStorage.getItem('x-organization') || 'BancoCentral';
  },

  emitirTCA: async (codigoCAR: string, cpfCNPJHash: string) => {
    const res = await api.post('/tca', { codigoCAR, cpfCNPJHash });
    return res.data;
  },
  
  consultarMeusTCAs: async () => {
    const res = await api.get('/tca/meus');
    return res.data;
  },
    
  listarTodosTCAs: async () => {
    const res = await api.get('/tca/todos');
    return res.data;
  },

  consultarTCA: async (codigoCAR: string) => {
    const res = await api.get(`/tca/${codigoCAR}`);
    return res.data;
  },
    
  auditarTransacoes: async (codigoCAR: string) => {
    const res = await api.get(`/tca/${codigoCAR}/historico`);
    return res.data;
  },
    
  revalidarTCA: async (tcaOrigemID: string, cpfCNPJHash: string, codigoCAR: string) => {
    const res = await api.post('/tca/revalidar', { tcaOrigemID, cpfCNPJHash, codigoCAR });
    return res.data;
  },

  suspenderTCA: async (id: string, motivo: string) => {
    const res = await api.post(`/tca/${id}/suspender`, { motivo });
    return res.data;
  },
    
  reativarTCA: async (id: string, codigoCAR: string) => {
    const res = await api.post(`/tca/${id}/reativar`, { codigoCAR });
    return res.data;
  },
    
  finalizarTCA: async (id: string) => {
    const res = await api.post(`/tca/${id}/finalizar`);
    return res.data;
  },
};
