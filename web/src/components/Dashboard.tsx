import React from 'react';
import { Monitor } from '../api';
import { Card, Row, Col, Badge } from 'react-bootstrap';
import { FaArrowUp, FaArrowDown, FaClock } from 'react-icons/fa';
import { useTranslation } from 'react-i18next';
import TopBar from './TopBar';

interface DashboardProps {
  monitors: Monitor[];
  onSelectMonitor: (id: string) => void;
  searchTerm: string;
  onSearch: (term: string) => void;
}

const Dashboard: React.FC<DashboardProps> = ({ monitors, onSelectMonitor, searchTerm, onSearch }) => {
  const { t } = useTranslation();
  const upCount = monitors.filter(m => m.status === 'up').length;
  const downCount = monitors.filter(m => m.status === 'down').length;
  const unknownCount = monitors.length - upCount - downCount;
  
  const total = monitors.length;
  const upPercent = total > 0 ? (upCount / total) * 100 : 0;
  const downPercent = total > 0 ? (downCount / total) * 100 : 0;
  const unknownPercent = total > 0 ? (unknownCount / total) * 100 : 0;

  return (
    <div className="animate-fade-in">
      <div style={{
        position: 'sticky',
        top: 0,
        zIndex: 100,
        backgroundColor: 'var(--bg-body)',
        padding: '10px 20px 0 20px'
      }}>
        <div className="d-flex justify-content-between align-items-center mb-4">
          <h2 className="mb-0">{t('dashboard')}</h2>
          <TopBar searchTerm={searchTerm} onSearch={onSearch} />
        </div>
        
        <Row className="mb-4 g-3">
          <Col md={4}>
            <Card className="wave-card h-100">
              <div className="wave-wrapper wave-status-up" style={{ height: `${upPercent}%` }}>
                  <div className="wave-block"></div>
                  <div className="wave-surface wave-surface-1"></div>
                  <div className="wave-surface wave-surface-2"></div>
                  <div className="wave-surface wave-surface-3"></div>
              </div>
              <Card.Body className="d-flex align-items-center justify-content-between position-relative" style={{ zIndex: 1 }}>
                <div>
                  <h6 className="text-secondary">{t('status.up')}</h6>
                  <h2 className="mb-0" style={{ color: 'var(--text-primary)' }}>{upCount}</h2>
                </div>
                <FaArrowUp size={30} className="text-success opacity-50" />
              </Card.Body>
            </Card>
          </Col>
          <Col md={4}>
            <Card className="wave-card h-100">
              <div className="wave-wrapper wave-status-down" style={{ height: `${downPercent}%` }}>
                  <div className="wave-block"></div>
                  <div className="wave-surface wave-surface-1"></div>
                  <div className="wave-surface wave-surface-2"></div>
                  <div className="wave-surface wave-surface-3"></div>
              </div>
              <Card.Body className="d-flex align-items-center justify-content-between position-relative" style={{ zIndex: 1 }}>
                <div>
                  <h6 className="text-secondary">{t('status.down')}</h6>
                  <h2 className="mb-0" style={{ color: 'var(--text-primary)' }}>{downCount}</h2>
                </div>
                <FaArrowDown size={30} className="text-danger opacity-50" />
              </Card.Body>
            </Card>
          </Col>
          <Col md={4}>
            <Card className="wave-card h-100">
              <div className="wave-wrapper wave-status-pending" style={{ height: `${unknownPercent}%` }}>
                  <div className="wave-block"></div>
                  <div className="wave-surface wave-surface-1"></div>
                  <div className="wave-surface wave-surface-2"></div>
                  <div className="wave-surface wave-surface-3"></div>
              </div>
              <Card.Body className="d-flex align-items-center justify-content-between position-relative" style={{ zIndex: 1 }}>
                <div>
                  <h6 className="text-secondary">{t('status.pending')}</h6>
                  <h2 className="mb-0" style={{ color: 'var(--text-primary)' }}>{unknownCount}</h2>
                </div>
                <FaClock size={30} className="text-warning opacity-50" />
              </Card.Body>
            </Card>
          </Col>
        </Row>
      </div>

      <div style={{ padding: '0 20px 20px 20px' }}>
        <h4 className="mb-3">{t('monitors')}</h4>
        <Row className="g-3">
          {monitors.map(m => (
            <Col md={6} lg={4} key={m.id}>
              <div className="kuba-card h-100 cursor-pointer" onClick={() => onSelectMonitor(m.id)} style={{cursor: 'pointer'}}>
                <div className="d-flex flex-column h-100">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <div className="d-flex align-items-center">
                        <span className={`status-dot ${m.status === 'up' ? 'up' : m.status === 'down' ? 'down' : 'pending'}`}></span>
                        <h5 className="mb-0 text-truncate">{m.name}</h5>
                    </div>
                    <Badge bg={m.type === 'container' ? 'info' : 'secondary'}>
                      {m.type === 'http' ? t('monitor.types.http') : t('monitor.types.container')}
                    </Badge>
                  </div>
                  <div className="text-secondary small text-truncate mb-2" style={{ color: 'var(--text-secondary)', minHeight: '1.5em' }}>
                    {m.url || '\u00A0'}
                  </div>
                  <div className="mt-auto">
                      {m.lastCheck && (
                          <div className="text-secondary small">
                            {t('detail.lastCheck')}: {new Date(m.lastCheck).toLocaleString()}
                          </div>
                      )}
                  </div>
                </div>
              </div>
            </Col>
          ))}
        </Row>
      </div>
    </div>
  );
};

export default Dashboard;
