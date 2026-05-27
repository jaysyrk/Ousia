import { useState, useEffect } from 'react';

export default function Topology() {
  const [services, setServices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchTopology = async () => {
      try {
        const res = await fetch('/api/admin/mesh/services');
        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
        const data = await res.json();
        setServices(data.services || []);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchTopology();
    const interval = setInterval(fetchTopology, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading) return <div style={{ color: 'var(--text-secondary)' }}>Loading topology...</div>;
  if (error) return <div style={{ color: 'var(--danger)' }}>Error loading topology: {error}</div>;

  return (
    <div>
      {services.length === 0 ? (
        <p style={{ color: 'var(--text-secondary)' }}>No services registered.</p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
          {services.map((svc) => (
            <div
              key={svc.service_id}
              style={{
                background: 'rgba(0,0,0,0.2)',
                padding: '1rem',
                borderRadius: '8px',
                border: '1px solid var(--glass-border)'
              }}
            >
              <h4 style={{ color: 'var(--accent-teal)', marginBottom: '0.5rem' }}>{svc.service_id}</h4>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                {svc.instances.map((inst) => (
                  <div key={inst.instance_id} style={{ fontSize: '0.9rem', display: 'flex', justifyContent: 'space-between' }}>
                    <span>{inst.instance_id}</span>
                    <span style={{ color: 'var(--text-secondary)' }}>
                      {inst.address}:{inst.port}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
