import { useState } from 'react'
import Topology from './components/Topology'
import Metrics from './components/Metrics'

function App() {
  const [activeTab, setActiveTab] = useState('topology')

  return (
    <div className="container fade-in">
      <h1>Ousia Control Plane</h1>
      <p style={{ marginBottom: '2rem', color: 'var(--text-secondary)' }}>
        Welcome to the next-generation service mesh dashboard.
      </p>

      <div style={{ marginBottom: '2rem', display: 'flex', gap: '1rem' }}>
        <button 
          className="btn-primary" 
          onClick={() => setActiveTab('topology')}
          style={{ opacity: activeTab === 'topology' ? 1 : 0.6 }}
        >
          Network Topology
        </button>
        <button 
          className="btn-primary" 
          onClick={() => setActiveTab('metrics')}
          style={{ opacity: activeTab === 'metrics' ? 1 : 0.6 }}
        >
          Live Metrics
        </button>
      </div>

      <div className="glass-panel" style={{ minHeight: '400px' }}>
        {activeTab === 'topology' ? (
          <div>
            <h3 style={{ marginBottom: '1rem' }}>Topology</h3>
            <Topology />
          </div>
        ) : (
          <div>
            <h3 style={{ marginBottom: '1rem' }}>System Metrics</h3>
            <Metrics />
          </div>
        )}
      </div>
    </div>
  )
}

export default App
