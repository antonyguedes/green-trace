

export const ScoreBadge = ({ score }: { score: number }) => {
  let color = '#10b981'; // default green
  let glowColor = 'rgba(16, 185, 129, 0.2)';
  if (score < 50) { 
    color = '#ef4444'; 
    glowColor = 'rgba(239, 68, 68, 0.2)'; 
  }
  else if (score < 80) { 
    color = '#eab308'; 
    glowColor = 'rgba(234, 179, 8, 0.2)'; 
  }

  return (
    <div style={{
      padding: '4px 10px',
      borderRadius: '20px',
      border: `1px solid ${color}`,
      background: glowColor,
      color: color,
      fontWeight: 600,
      fontSize: '0.8rem',
      display: 'inline-flex',
      alignItems: 'center',
      gap: '6px',
      boxShadow: `0 0 10px ${glowColor}`
    }}>
      <div style={{ width: 6, height: 6, borderRadius: '50%', backgroundColor: color, boxShadow: `0 0 5px ${color}` }} />
      Score: {score}/100
    </div>
  )
};
