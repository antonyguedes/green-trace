import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { tcaApi } from './api';
import { TcaCard } from './components/TcaCard';
import { Leaf, Database, Building2, ShieldCheck, Activity } from 'lucide-react';

function App() {
  const queryClient = useQueryClient();
  const [org, setOrgState] = useState(tcaApi.getOrg());
  const [activeTab, setActiveTab] = useState('portfolio');
  const [carInput, setCarInput] = useState('CAR-123-LIMPO');

  const { data: tcas, isLoading } = useQuery({
    queryKey: ['tcas', org],
    queryFn: () => org === 'BancoCentral' ? tcaApi.listarTodosTCAs() : tcaApi.consultarMeusTCAs(),
  });

  const changeOrg = (newOrg: string) => {
    tcaApi.setOrg(newOrg);
    setOrgState(newOrg);
    queryClient.invalidateQueries({ queryKey: ['tcas'] });
  };

  const emitirMutation = useMutation({
    mutationFn: () => tcaApi.emitirTCA(carInput, 'HASH-123'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tcas'] });
      setActiveTab('portfolio');
    }
  });

  const suspenderMutation = useMutation({
    mutationFn: (tca: any) => tcaApi.suspenderTCA(tca.id, "Monitoramento automático acusou irregularidade"),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tcas'] })
  });

  const reativarMutation = useMutation({
    mutationFn: (tca: any) => tcaApi.reativarTCA(tca.id, tca.codigoCAR),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tcas'] })
  });

  const revalidarMutation = useMutation({
    mutationFn: (tca: any) => tcaApi.revalidarTCA(tca.id, tca.cpfCNPJHash, tca.codigoCAR),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tcas'] })
  });

  const handleAction = (action: string, tca: any) => {
    if (action === 'suspender') suspenderMutation.mutate(tca);
    if (action === 'reativar') reativarMutation.mutate(tca);
    if (action === 'revalidar') revalidarMutation.mutate(tca);
  };

  return (
    <div style={{ display: 'flex', minHeight: '100vh', padding: '32px', gap: '40px', maxWidth: '1400px', margin: '0 auto' }}>
      
      {/* Sidebar / Org Selector */}
      <div style={{ width: '280px', flexShrink: 0, display: 'flex', flexDirection: 'column', gap: '32px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{ width: 44, height: 44, borderRadius: '12px', background: 'var(--brand-glow)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--brand-primary)', boxShadow: '0 0 20px var(--brand-glow)' }}>
            <Leaf size={24} />
          </div>
          <h1 style={{ margin: 0, fontSize: '1.8rem', fontWeight: 700 }} className="text-gradient">GreenTrace</h1>
        </div>

        <div className="glass-panel" style={{ padding: '24px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <h2 style={{ fontSize: '0.85rem', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em', fontWeight: 600 }}>Identidade Ativa (Nó gRPC)</h2>
          
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {['BancoCentral', 'InstFinA', 'InstFinB', 'OrgAmbiental'].map(o => (
              <button 
                key={o}
                onClick={() => changeOrg(o)}
                style={{
                  width: '100%',
                  padding: '14px',
                  borderRadius: '10px',
                  border: org === o ? '1px solid var(--border-focus)' : '1px solid var(--border-light)',
                  background: org === o ? 'rgba(16, 185, 129, 0.1)' : 'transparent',
                  color: org === o ? 'var(--brand-primary)' : 'var(--text-main)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '10px',
                  cursor: 'pointer',
                  transition: 'all 0.2s',
                  textAlign: 'left',
                  fontWeight: org === o ? 600 : 500,
                  boxShadow: org === o ? '0 0 15px rgba(16, 185, 129, 0.1)' : 'none'
                }}
              >
                {o === 'BancoCentral' ? <Database size={18} /> : o === 'OrgAmbiental' ? <Activity size={18} /> : <Building2 size={18} />}
                {o}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '32px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', paddingBottom: '16px', borderBottom: '1px solid var(--border-light)' }}>
          <h2 style={{ fontSize: '1.8rem', fontWeight: 600, letterSpacing: '-0.02em' }}>
            {org === 'BancoCentral' ? 'Visão Regulador (State DB)' : 'Gestão de Conformidade (TCA)'}
          </h2>
          
          <div style={{ display: 'flex', gap: '8px', background: 'rgba(0,0,0,0.3)', padding: '6px', borderRadius: '12px', border: '1px solid var(--border-light)' }}>
            <button 
              onClick={() => setActiveTab('portfolio')}
              style={{ padding: '8px 20px', borderRadius: '8px', background: activeTab === 'portfolio' ? 'rgba(255,255,255,0.1)' : 'transparent', color: activeTab === 'portfolio' ? '#fff' : 'var(--text-muted)', border: 'none', cursor: 'pointer', fontWeight: 500, transition: 'all 0.2s' }}
            >
              Portfólio de Ativos
            </button>
            <button 
              onClick={() => setActiveTab('emitir')}
              style={{ padding: '8px 20px', borderRadius: '8px', background: activeTab === 'emitir' ? 'rgba(255,255,255,0.1)' : 'transparent', color: activeTab === 'emitir' ? '#fff' : 'var(--text-muted)', border: 'none', cursor: 'pointer', fontWeight: 500, transition: 'all 0.2s' }}
            >
              Nova Emissão
            </button>
          </div>
        </div>

        {activeTab === 'portfolio' && (
          <div className="animate-fade-in" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(400px, 1fr))', gap: '24px' }}>
            {isLoading ? (
              <div style={{ display: 'flex', gap: '12px', alignItems: 'center', color: 'var(--text-muted)' }}>
                 <Activity className="animate-spin" /> Carregando do Ledger...
              </div>
            ) : tcas && tcas.length > 0 ? (
              tcas.map((item: any) => {
                // BC view has slightly different nested property
                const tca = item.Record || item; 
                return <TcaCard key={tca.id} tca={tca} onAction={handleAction} />
              })
            ) : (
              <div style={{ padding: '40px', background: 'rgba(255,255,255,0.02)', borderRadius: '16px', border: '1px dashed var(--border-light)', textAlign: 'center', color: 'var(--text-muted)' }}>
                Nenhum TCA documentado na rede.
              </div>
            )}
          </div>
        )}

        {activeTab === 'emitir' && (
          <div className="animate-fade-in glass-panel" style={{ padding: '40px', maxWidth: '640px' }}>
            <h3 style={{ fontSize: '1.5rem', fontWeight: 600, marginBottom: '8px' }}>Gerar Token de Conformidade</h3>
            <p style={{ color: 'var(--text-muted)', marginBottom: '32px' }}>Os dados serão processados via Smart Contract (Chaincode) após validação do oráculo governamental.</p>
            
            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
              <div>
                <label style={{ display: 'block', fontSize: '0.9rem', color: 'var(--text-muted)', marginBottom: '10px', fontWeight: 500 }}>Identificação Rural (Código CAR)</label>
                <select 
                  value={carInput}
                  onChange={(e) => setCarInput(e.target.value)}
                  style={{ width: '100%', padding: '16px', borderRadius: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', fontSize: '1.05rem', outline: 'none', transition: 'border 0.2s' }}
                >
                  <option value="CAR-100-LIMPO">CAR-100-LIMPO (Cenário: 100pts Conformidade)</option>
                  <option value="CAR-200-DESMATADO">CAR-200-DESMATADO (Erro: Bloqueio PRODES)</option>
                  <option value="CAR-300-EMBARGADO">CAR-300-EMBARGADO (Erro: Embargo IBAMA)</option>
                  <option value="CAR-400-MEDIO">CAR-400-MEDIO (Cenário: RL Irregular - 60pts)</option>
                </select>
              </div>

              <div style={{ padding: '20px', background: 'rgba(16, 185, 129, 0.05)', borderRadius: '12px', border: '1px dashed var(--border-focus)' }}>
                <div style={{ display: 'flex', gap: '12px', alignItems: 'center', color: 'var(--brand-primary)', marginBottom: '12px' }}>
                  <ShieldCheck size={22} />
                  <span style={{ fontWeight: 600, fontSize: '1.05rem' }}>Oráculo Integrado Ativado</span>
                </div>
                <p style={{ margin: 0, fontSize: '0.95rem', color: 'var(--text-muted)', lineHeight: '1.5' }}>
                  As transações na rede Fabric irão buscar estado atual no IBAMA, SICAR, INPE e MTE de forma determinística antes de gravar no Ledger.
                </p>
              </div>

              <button 
                className="btn-primary" 
                onClick={() => emitirMutation.mutate()}
                disabled={emitirMutation.isPending}
                style={{ justifyContent: 'center', marginTop: '16px', padding: '16px', fontSize: '1.1rem', letterSpacing: '0.02em' }}
              >
                {emitirMutation.isPending ? 'Propagando transação...' : 'Gravar TCA (Invocar Chaincode)'}
              </button>
            </div>
          </div>
        )}

      </div>
    </div>
  )
}

export default App
