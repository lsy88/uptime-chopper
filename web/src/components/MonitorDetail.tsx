import React, { useState, useEffect, useRef } from 'react';
import { 
    Monitor, 
    Container, 
    deleteMonitor, 
    NotificationWebhook,
    getNotifications,
    getContainerLogs,
    startContainer,
    stopContainer,
    restartContainer,
    setRestartPolicy
} from '../api';
import { Badge, Button, ButtonGroup, Spinner, Form, Alert, Row, Col } from 'react-bootstrap';
import { FaPlay, FaStop, FaRedo, FaTerminal, FaBox, FaClock, FaEdit, FaTrash } from 'react-icons/fa';
import { useTranslation } from 'react-i18next';
import AddMonitorModal from './AddMonitorModal';

interface MonitorDetailProps {
  monitor: Monitor;
  containers: Container[];
  onRefresh: () => void;
}

const MonitorDetail: React.FC<MonitorDetailProps> = ({ monitor, containers, onRefresh }) => {
  const { t } = useTranslation();
  const [logs, setLogs] = useState<string>('');
  const [loadingLogs, setLoadingLogs] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showEditModal, setShowEditModal] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [notifications, setNotifications] = useState<NotificationWebhook[]>([]);
  
  // For container monitors, find the linked container info if available
  // In a real app, we might need a better way to link, currently matching by name or ID if possible?
  // But wait, the monitor config has `containerName`.
  const linkedContainer = containers.find(c => c.names?.some(n => n.replace('/', '') === monitor.containerName));
  const containerId = linkedContainer?.id;

  const logsEndRef = useRef<HTMLDivElement>(null);

  const loadNotifications = async () => {
    try {
      const data = await getNotifications();
      setNotifications(data || []);
    } catch (err) {
      console.error("Failed to load notifications", err);
    }
  };

  const fetchLogs = async () => {
    if (!containerId) return;
    setLoadingLogs(true);
    try {
      const data = await getContainerLogs(containerId);
      setLogs(data);
      // Auto scroll to bottom
      setTimeout(() => {
        logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
      }, 100);
    } catch (err) {
      console.error(err);
    } finally {
      setLoadingLogs(false);
    }
  };

  useEffect(() => {
    loadNotifications();
    if (monitor.type === 'container' && containerId) {
      fetchLogs();
    } else {
        setLogs('');
    }
  }, [monitor.id, containerId]);

  const handleAction = async (action: 'start' | 'stop' | 'restart') => {
    if (!containerId) return;
    setActionLoading(true);
    setError(null);
    try {
      if (action === 'start') await startContainer(containerId);
      if (action === 'stop') await stopContainer(containerId);
      if (action === 'restart') await restartContainer(containerId);
      await new Promise(r => setTimeout(r, 1000)); // Wait a bit
      onRefresh(); // Refresh state
      fetchLogs(); // Refresh logs
    } catch (err: any) {
      setError(err.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  const [targetPolicy, setTargetPolicy] = useState('no');

  const handleRestartPolicy = async (policy: string) => {
      setTargetPolicy(policy);
      if (!containerId) return;
      try {
          await setRestartPolicy(containerId, policy);
          onRefresh();
      } catch (err: any) {
          setError("Failed to set restart policy: " + err.message);
      }
  };

  const handleDelete = async () => {
    if (window.confirm(t('confirmDelete', { defaultValue: 'Are you sure you want to delete this monitor?' }))) {
      setDeleting(true);
      try {
        await deleteMonitor(monitor.id);
        // Force refresh which will trigger parent to see monitor is gone
        // But parent needs to know to clear selection. 
        // We can just call onRefresh, and parent App logic should handle if selected ID is missing?
        // Actually App.tsx currently: selectedMonitor = monitors.find(m => m.id === selectedMonitorId);
        // If not found, it might show Dashboard. Let's verify.
        onRefresh(); 
      } catch (err: any) {
        setError("Failed to delete monitor: " + err.message);
        setDeleting(false);
      }
    }
  };

  return (
    <div className="animate-fade-in" style={{ padding: '20px' }}>
      <div className="d-flex align-items-center justify-content-between mb-4">
        <div>
            <h2 className="mb-1 d-flex align-items-center">
                <span className={`status-dot ${monitor.status === 'up' ? 'up' : monitor.status === 'down' ? 'down' : 'pending'} me-3`}></span>
                {monitor.name}
                <sup className="ms-2">
                    <Badge bg={monitor.type === 'container' ? 'info' : 'secondary'} style={{ fontSize: '0.4em' }}>
                        {monitor.type === 'http' ? t('monitor.types.http') : t('monitor.types.container')}
                    </Badge>
                </sup>
            </h2>
        </div>
        <div className="d-flex align-items-center gap-3">
            <Button variant="outline-primary" size="sm" onClick={() => setShowEditModal(true)}>
                <FaEdit className="me-2" /> {t('monitor.edit')}
            </Button>
            <Button variant="outline-danger" size="sm" onClick={handleDelete} disabled={deleting}>
                <FaTrash className="me-2" /> {t('monitor.delete')}
            </Button>
            <div className="text-end border-start ps-3 ms-2">
                <div className="h4 mb-0 text-primary">
                    {monitor.status === 'up' ? '100%' : monitor.status === 'down' ? '0%' : '--'}
                </div>
                <div className="small text-secondary">{t('detail.uptime')} (24h)</div>
            </div>
        </div>
      </div>

      {error && <Alert variant="danger" onClose={() => setError(null)} dismissible>{error}</Alert>}

      {monitor.type === 'container' && linkedContainer && (
        <div className="kuba-card mb-4">
            <div className="kuba-card-header">
                <span><FaBox className="me-2" /> {t('monitor.container')}</span>
                <Badge bg={linkedContainer.state === 'running' ? 'success' : 'danger'}>{linkedContainer.state}</Badge>
            </div>
            
            <div className="d-flex flex-wrap gap-3 align-items-center mb-4">
                <ButtonGroup>
                    <Button 
                        variant="success" 
                        onClick={() => handleAction('start')} 
                        disabled={actionLoading || linkedContainer.state === 'running'}
                    >
                        <FaPlay className="me-2" /> {t('monitor.start')}
                    </Button>
                    <Button 
                        variant="danger" 
                        onClick={() => handleAction('stop')} 
                        disabled={actionLoading || linkedContainer.state !== 'running'}
                    >
                        <FaStop className="me-2" /> {t('monitor.stop')}
                    </Button>
                    <Button 
                        variant="warning" 
                        onClick={() => handleAction('restart')} 
                        disabled={actionLoading}
                    >
                        <FaRedo className="me-2" /> {t('monitor.restart')}
                    </Button>
                </ButtonGroup>

                <div className="d-flex align-items-center ms-auto">
                    <span className="me-2 text-secondary">Restart Policy:</span>
                    <Form.Select 
                        size="sm" 
                        style={{width: 'auto'}} 
                        value={targetPolicy}
                        onChange={(e) => handleRestartPolicy(e.target.value)}
                        className="bg-card text-primary border-secondary"
                    >
                        <option value="no">No</option>
                        <option value="always">Always</option>
                        <option value="on-failure">On Failure</option>
                        <option value="unless-stopped">Unless Stopped</option>
                    </Form.Select>
                </div>
            </div>

            <div className="d-flex align-items-center justify-content-between mb-2">
                <h6 className="mb-0 text-secondary"><FaTerminal className="me-2" /> {t('detail.logs')}</h6>
                <Button size="sm" variant="outline-secondary" onClick={fetchLogs} disabled={loadingLogs}>
                    {loadingLogs ? <Spinner size="sm" animation="border" /> : <FaRedo />} {t('monitor.refresh')}
                </Button>
            </div>
            <div className="log-viewer">
                {logs ? <pre className="m-0 text-break">{logs}</pre> : <div className="text-center text-muted mt-5">No logs available</div>}
                <div ref={logsEndRef} />
            </div>
        </div>
      )}

      <div className="kuba-card">
          <div className="kuba-card-header">{t('monitor.details')}</div>
          <Row>
              <Col md={6}>
                  <div className="mb-3">
                      <label className="text-secondary small">{monitor.type === 'container' ? t('monitor.container') : t('monitor.url')}</label>
                      <div className="text-break">{monitor.type === 'container' ? monitor.containerName : monitor.url}</div>
                  </div>
                  <div className="mb-3">
                      <label className="text-secondary small">{t('monitor.interval')}</label>
                      <div>{monitor.intervalSeconds}s</div>
                  </div>
              </Col>
              <Col md={6}>
                  <div className="mb-3">
                      <label className="text-secondary small">{t('detail.lastCheck')}</label>
                      <div>{monitor.lastCheck ? new Date(monitor.lastCheck).toLocaleString() : 'Never'}</div>
                  </div>
                  <div className="mb-3">
                      <label className="text-secondary small">{t('monitor.notifications')}</label>
                      <div>
                          {monitor.notifyWebhookIds && monitor.notifyWebhookIds.length > 0 ? (
                              <div className="d-flex flex-wrap gap-1 mt-1">
                                  {monitor.notifyWebhookIds.map(id => {
                                      const notif = notifications.find(n => n.id === id || n.name === id);
                                      return (
                                          <Badge key={id} bg="secondary" className="fw-normal">
                                              {notif ? notif.name : id}
                                          </Badge>
                                      );
                                  })}
                              </div>
                          ) : (
                              <div 
                                className="text-secondary small text-decoration-underline" 
                                style={{ cursor: 'pointer' }}
                                onClick={() => setShowEditModal(true)}
                            >
                                {t('monitor.noNotifications')}
                            </div>
                          )}
                      </div>
                  </div>
              </Col>
          </Row>
      </div>

      <AddMonitorModal 
        show={showEditModal}
        onHide={() => setShowEditModal(false)}
        onSaved={onRefresh}
        initialData={monitor}
      />
    </div>
  );
};

export default MonitorDetail;
