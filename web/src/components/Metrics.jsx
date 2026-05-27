import { useState, useEffect } from 'react';

export default function Metrics() {
  const [stats, setStats] = useState({ gateway: null, sidecars: {} });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const res = await fetch('/api/admin/stats');
        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
        const data = await res.json();
        setStats({
          gateway: data.gateway,
          sidecars: data.sidecars || {}
        });
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
    const interval = setInterval(fetchStats, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading) return <div style={{ color: 'var(--text-secondary)' }}>Loading metrics...</div>;
  if (error) return <div style={{ color: 'var(--danger)' }}>Error loading metrics: {error}</div>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
      <div
        style={{
          background: 'rgba(0,0,0,0.2)',
          padding: '1rem',
          borderRadius: '8px',
          border: '1px solid var(--glass-border)'
        }}
      >
        <h4 style={{ color: 'var(--accent-cyan)', marginBottom: '0.5rem' }}>Gateway Traffic</h4>
        <pre style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', overflowX: 'auto' }}>
          {stats.gateway ? JSON.stringify(stats.gateway, null, 2) : 'No gateway stats available'}
        </pre>
      </div>

      {Object.entries(stats.sidecars).map(([serviceId, instances]) => (
        <div key={serviceId}>
          <h5 style={{ marginTop: '1rem', marginBottom: '0.5rem' }}>Service: {serviceId}</h5>
          {Object.entries(instances).map(([instanceId, statStr]) => (
            <div
              key={instanceId}
              style={{
                background: 'rgba(0,0,0,0.1)',
                padding: '0.5rem 1rem',
                borderRadius: '8px',
                border: '1px solid var(--glass-border)',
                marginBottom: '0.5rem'
              }}
            >
              <div style={{ fontSize: '0.8rem', color: 'var(--accent-teal)' }}>{instanceId}</div>
              <pre style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', overflowX: 'auto', margin: 0 }}>
                {statStr}
              </pre>
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}
