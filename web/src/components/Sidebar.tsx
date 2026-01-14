import React from 'react';
import { Monitor } from '../api';
import { FaPlus, FaSearch } from 'react-icons/fa';
import { Badge, Form, Button } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

interface SidebarProps {
  monitors: Monitor[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  onAdd: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ monitors, selectedId, onSelect, onAdd }) => {
  const { t } = useTranslation();

  const getStatusClass = (status: string) => {
    switch (status) {
      case 'up': return 'up';
      case 'down': return 'down';
      default: return 'pending';
    }
  };

  return (
    <div className="kuba-sidebar">
      <div className="p-3 border-bottom border-secondary">
        <h4 className="text-primary mb-0 d-flex align-items-center cursor-pointer" onClick={() => onSelect(null)} style={{cursor: 'pointer'}}>
          <img src="/logo.svg" alt="logo" height="42" className="me-2" />
          Uptime Chopper
        </h4>
      </div>
      
      <div className="p-3">
        <div className="d-flex gap-2 mb-3">
            <Button variant="success" className="w-100 d-flex align-items-center justify-content-center" onClick={onAdd}>
                <FaPlus className="me-2" /> {t('addMonitor')}
            </Button>
        </div>
      </div>

      <div className="flex-grow-1 overflow-auto">
        <div 
            className={`monitor-list-item ${selectedId === null ? 'active' : ''}`}
            onClick={() => onSelect(null)} 
        >
            <span className="fw-bold">{t('dashboard')}</span>
        </div>
        
        <div className="text-secondary text-uppercase fs-7 px-3 mt-3 mb-2 fw-bold" style={{fontSize: '0.8rem'}}>{t('monitors')}</div>
        
        {monitors.map(m => (
          <div 
            key={m.id} 
            className={`monitor-list-item ${selectedId === m.id ? 'active' : ''}`}
            onClick={() => onSelect(m.id)}
          >
            <div className="overflow-hidden">
                <div className="d-flex align-items-center">
                    <span className={`status-dot ${getStatusClass(m.status)}`}></span>
                    <div className="fw-bold text-truncate">{m.name}</div>
                </div>
                {m.url && (
                    <div className="small text-truncate" style={{fontSize: '0.75rem', paddingLeft: '20px', color: 'var(--text-secondary)'}}>
                        {m.url}
                    </div>
                )}
            </div>
            {m.status === 'down' && <Badge bg="danger" pill>!</Badge>}
          </div>
        ))}
      </div>

    </div>
  );
};

export default Sidebar;
