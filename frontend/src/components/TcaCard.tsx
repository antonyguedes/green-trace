import { ScoreBadge } from './ScoreBadge';
import { ShieldAlert, Info } from 'lucide-react';

export const TcaCard = ({ tca, onAction }: any) => {
    const statusColor = tca.status === 'ATIVO' ? '#10b981' : 
                        tca.status === 'SUSPENSO' ? '#eab308' : 
                        tca.status === 'NEGADO' ? '#ef4444' : '#3b82f6';
                        
    return (
        <div className="glass-panel" style={{ padding: '24px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                    <h3 style={{ margin: 0, fontSize: '1.2rem', fontWeight: 600 }}>{tca.id}</h3>
                    <p style={{ margin: 0, color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '4px' }}>CAR: {tca.codigoCAR}</p>
                </div>
                <div style={{ 
                    padding: '6px 12px', 
                    borderRadius: '8px', 
                    border: `1px solid ${statusColor}40`, 
                    color: statusColor, 
                    fontWeight: 700, 
                    fontSize: '0.8rem', 
                    background: `${statusColor}15`,
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px'
                }}>
                    <div style={{ width: 6, height: 6, borderRadius: '50%', backgroundColor: statusColor }} />
                    {tca.status}
                </div>
            </div>
            
            <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap', alignItems: 'center' }}>
                 <ScoreBadge score={tca.scoreConformidade} />
                 
                 {tca.impedimentos?.desmatamentoIlegal && (
                     <div style={{ display: 'flex', alignItems: 'center', gap: '6px', color: '#ef4444', fontSize: '0.85rem', background: 'rgba(239, 68, 68, 0.1)', padding: '4px 10px', borderRadius: '12px' }}>
                         <ShieldAlert size={14} /> Desmatamento
                     </div>
                 )}
                 {tca.impedimentos?.embargoIBAMA && (
                     <div style={{ display: 'flex', alignItems: 'center', gap: '6px', color: '#ef4444', fontSize: '0.85rem', background: 'rgba(239, 68, 68, 0.1)', padding: '4px 10px', borderRadius: '12px' }}>
                         <ShieldAlert size={14} /> Embargo Ativo
                     </div>
                 )}
            </div>
            
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px', marginTop: '8px', borderTop: '1px solid var(--border-light)', paddingTop: '16px' }}>
                <div>
                    <span style={{ color: 'var(--text-muted)', fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>Emissão</span>
                    <div style={{ fontSize: '0.95rem', fontWeight: 500, marginTop: '4px' }}>{new Date(tca.dataEmissao).toLocaleDateString('pt-BR')}</div>
                </div>
                <div>
                    <span style={{ color: 'var(--text-muted)', fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>Validade</span>
                    <div style={{ fontSize: '0.95rem', fontWeight: 500, marginTop: '4px' }}>{new Date(tca.dataValidade).toLocaleDateString('pt-BR')}</div>
                </div>
            </div>

            {tca.historicoSuspensoes && tca.historicoSuspensoes.length > 0 && (
                <div style={{ marginTop: '12px', padding: '12px', background: 'rgba(234, 179, 8, 0.05)', borderRadius: '8px', border: '1px dashed rgba(234, 179, 8, 0.2)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', color: '#eab308', fontSize: '0.8rem', fontWeight: 600, marginBottom: '8px' }}>
                        <Info size={14} /> Histórico de Suspensões ({tca.historicoSuspensoes.length})
                    </div>
                    {tca.historicoSuspensoes.slice(-1).map((susp: any, i: number) => (
                        <div key={i} style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
                            Última: {susp.motivo} ({new Date(susp.dataSuspenso).toLocaleDateString('pt-BR')})
                        </div>
                    ))}
                </div>
            )}
            
            {onAction && (
                <div style={{ marginTop: '8px', display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
                    {tca.status === 'ATIVO' && (
                        <button className="btn-secondary" onClick={() => onAction('suspender', tca)}>
                            Suspender
                        </button>
                    )}
                    {tca.status === 'SUSPENSO' && (
                        <button className="btn-primary" onClick={() => onAction('reativar', tca)}>
                            Reativar
                        </button>
                    )}
                    {(tca.status === 'ATIVO' || tca.status === 'SUSPENSO') && (
                        <button className="btn-secondary" onClick={() => onAction('revalidar', tca)}>
                            Revalidar
                        </button>
                    )}
                </div>
            )}
        </div>
    )
}
