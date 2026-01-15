import React from 'react';
import { Monitor } from '../api';
import { Card, Row, Col, Badge } from 'react-bootstrap';
import { FaArrowUp, FaArrowDown, FaClock } from 'react-icons/fa';
import { useTranslation } from 'react-i18next';
import TopBar from './TopBar';
import WaveCanvas from './WaveCanvas';

interface DashboardProps {
  monitors: Monitor[];
  onSelectMonitor: (id: string) => void;
  searchTerm: string;
  onSearch: (term: string) => void;
}

const WaveWrapper = ({ percent, colorRgb }: { percent: number, colorRgb: string }) => (
  <div 
    style={{ 
      position: 'absolute', 
      bottom: 0, 
      left: 0, 
      width: '100%', 
      height: `${percent}%`, 
      transition: 'height 1s cubic-bezier(0.4, 0, 0.2, 1)',
      zIndex: 0,
      opacity: percent <= 0 ? 0 : 1
    }}
  >
    <div style={{ position: 'absolute', top: '-25px', left: 0, width: '100%', height: '40px' }}>
      <WaveCanvas color={colorRgb} height={40} />
    </div>
    <div style={{ width: '100%', height: '100%', backgroundColor: `rgba(${colorRgb}, 1)` }}></div>
  </div>
);

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
            <Card className="wave-card h-100 overflow-hidden border-0 shadow-sm" style={{ backgroundColor: 'var(--bg-card)' }}>
              <WaveWrapper percent={upPercent} colorRgb="92, 221, 139" />
              <Card.Body className="d-flex align-items-center justify-content-between position-relative" style={{ zIndex: 1 }}>
                <div>
                  <h6 className="text-secondary" style={{ color: upPercent > 50 ? '#111' : 'inherit' }}>{t('status.up')}</h6>
                  <h2 className="mb-0" style={{ color: upPercent > 50 ? '#000' : 'var(--text-primary)' }}>{upCount}</h2>
                </div>
                <FaArrowUp size={30} className="text-success opacity-50" style={{ color: upPercent > 50 ? '#000 !important' : '' }} />
              </Card.Body>
            </Card>
          </Col>
          <Col md={4}>
            <Card className="wave-card h-100 overflow-hidden border-0 shadow-sm" style={{ backgroundColor: 'var(--bg-card)' }}>
              <WaveWrapper percent={downPercent} colorRgb="220, 53, 69" />
              <Card.Body className="d-flex align-items-center justify-content-between position-relative" style={{ zIndex: 1 }}>
                <div>
                  <h6 className="text-secondary" style={{ color: downPercent > 50 ? '#fff' : 'inherit' }}>{t('status.down')}</h6>
                  <h2 className="mb-0" style={{ color: downPercent > 50 ? '#fff' : 'var(--text-primary)' }}>{downCount}</h2>
                </div>
                <FaArrowDown size={30} className="text-danger opacity-50" style={{ color: downPercent > 50 ? '#fff !important' : '' }} />
              </Card.Body>
            </Card>
          </Col>
          <Col md={4}>
            <Card className="wave-card h-100 overflow-hidden border-0 shadow-sm" style={{ backgroundColor: 'var(--bg-card)' }}>
              <WaveWrapper percent={unknownPercent} colorRgb="255, 193, 7" />
              <Card.Body className="d-flex align-items-center justify-content-between position-relative" style={{ zIndex: 1 }}>
                <div>
                  <h6 className="text-secondary" style={{ color: unknownPercent > 50 ? '#000' : 'inherit' }}>{t('status.pending')}</h6>
                  <h2 className="mb-0" style={{ color: unknownPercent > 50 ? '#000' : 'var(--text-primary)' }}>{unknownCount}</h2>
                </div>
                <FaClock size={30} className="text-warning opacity-50" style={{ color: unknownPercent > 50 ? '#000 !important' : '' }} />
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
