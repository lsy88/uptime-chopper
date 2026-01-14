import React, { useEffect, useState } from 'react';
import { Monitor, Container, getMonitorsWithStatus, getContainers } from './api';
import Sidebar from './components/Sidebar';
import Dashboard from './components/Dashboard';
import MonitorDetail from './components/MonitorDetail';
import AddMonitorModal from './components/AddMonitorModal';
import TopBar from './components/TopBar';
import { Spinner } from 'react-bootstrap';

function App() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [containers, setContainers] = useState<Container[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedMonitorId, setSelectedMonitorId] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [filter, setFilter] = useState('');

  const refreshData = async () => {
    try {
      const [m, c] = await Promise.all([getMonitorsWithStatus(), getContainers()]);
      setMonitors(m);
      setContainers(c);
    } catch (error) {
      console.error('Failed to fetch data:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshData();
    const interval = setInterval(refreshData, 5000); // Auto refresh every 5s
    return () => clearInterval(interval);
  }, []);

  const handleAddMonitor = () => {
    setShowAddModal(true);
  };

  const selectedMonitor = monitors.find(m => m.id === selectedMonitorId);

  const filteredMonitors = monitors.filter(m => 
    m.name.toLowerCase().includes(filter.toLowerCase()) || 
    (m.url && m.url.toLowerCase().includes(filter.toLowerCase()))
  );

  if (loading && monitors.length === 0) {
    return (
      <div className="d-flex align-items-center justify-content-center vh-100" style={{backgroundColor: 'var(--bg-body)', color: 'var(--text-primary)'}}>
        <Spinner animation="border" variant="success" />
        <span className="ms-3">Loading Uptime Chopper...</span>
      </div>
    );
  }

  return (
    <div className="kuba-layout">
      <Sidebar 
        monitors={monitors} 
        selectedId={selectedMonitorId} 
        onSelect={setSelectedMonitorId}
        onAdd={handleAddMonitor}
      />
      
      <div className="kuba-main">
        {selectedMonitorId && selectedMonitor ? (
          <MonitorDetail 
            monitor={selectedMonitor} 
            containers={containers} 
            onRefresh={refreshData}
          />
        ) : (
          <Dashboard 
            monitors={filteredMonitors} 
            onSelectMonitor={setSelectedMonitorId} 
            searchTerm={filter}
            onSearch={setFilter}
          />
        )}
      </div>

      <AddMonitorModal 
        show={showAddModal}
        onHide={() => setShowAddModal(false)}
        onSaved={refreshData}
      />
    </div>
  );
}

export default App;
