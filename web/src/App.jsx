import { useState } from 'react'

function App() {
  const [count, setCount] = useState(0)

  return (
    <div className="container fade-in">
      <h1>Ousia Control Plane</h1>
      <p style={{ marginBottom: '2rem', color: 'var(--text-secondary)' }}>
        Welcome to the next-generation service mesh dashboard.
      </p>

      <div className="grid">
        <div className="glass-panel">
          <h3>Metrics</h3>
          <p>Real-time telemetry and performance statistics.</p>
          <div style={{ marginTop: '1rem' }}>
            <button className="btn-primary" onClick={() => setCount(count + 1)}>
              Ping Count: {count}
            </button>
          </div>
        </div>

        <div className="glass-panel">
          <h3>Topology</h3>
          <p>Service graph and network map.</p>
          <div style={{ marginTop: '1rem', color: 'var(--accent-teal)' }}>
            Status: Online
          </div>
        </div>

        <div className="glass-panel pulse" style={{ animationDuration: '3s' }}>
          <h3>System Health</h3>
          <p>All services are operating normally.</p>
          <div style={{ marginTop: '1rem', color: 'var(--success)' }}>
            100% Uptime
          </div>
        </div>
      </div>
    </div>
  )
}

export default App
