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
    setRestartPolicy,
    MonitorHistoryEntry,
    getMonitorHistory,
    pauseMonitor,
    resumeMonitor
} from '../api';
import { Badge, Button, ButtonGroup, Spinner, Form, Alert, Row, Col, Table } from 'react-bootstrap';
import { FaPlay, FaStop, FaRedo, FaTerminal, FaBox, FaClock, FaEdit, FaTrash, FaPause } from 'react-icons/fa';
import { useTranslation } from 'react-i18next';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
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
  const [history, setHistory] = useState<MonitorHistoryEntry[]>([]);
  const [loadingHistory, setLoadingHistory] = useState(false);
  
  // For container monitors, try to find the linked container info
  // If not found in the list, we'll fall back to using the containerName from the monitor config
  // This allows interacting with containers that might not be in the initial list (e.g. hidden or race condition)
  const linkedContainer = containers.find(c => 
    c.id === monitor.containerName || 
    c.id.startsWith(monitor.containerName || '') || 
    c.names?.some(n => n.replace('/', '') === monitor.containerName)
  );
  const containerId = linkedContainer?.id || monitor.containerName;

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

  const fetchHistory = async () => {
    try {
      const data = await getMonitorHistory(monitor.id);
      setHistory(data || []);
    } catch (err) {
      console.error("Failed to load history", err);
    }
  };

  useEffect(() => {
    loadNotifications();
    fetchHistory();
    if (monitor.type === 'container' && containerId) {
      fetchLogs();
    } else {
        setLogs('');
    }
  }, [monitor.id, containerId]);

  useEffect(() => {
      if (monitor.lastCheck) {
          fetchHistory();
      }
  }, [monitor.lastCheck]);

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

  const handlePauseToggle = async () => {
      try {
          if (monitor.isPaused) {
              await resumeMonitor(monitor.id);
          } else {
              await pauseMonitor(monitor.id);
          }
          onRefresh();
      } catch (err: any) {
          setError("Failed to toggle pause: " + err.message);
      }
  };

  const chartData = [...history].reverse().map(entry => ({
      time: new Date(entry.checkedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
      fullTime: new Date(entry.checkedAt).toLocaleString(),
      latency: entry.latencyMs,
      status: entry.status,
      message: entry.message
  }));

  const CustomTooltip = ({ active, payload, label }: any) => {
      if (active && payload && payload.length) {
          const data = payload[0].payload;
          return (
              <div className="custom-tooltip" style={{ 
                  backgroundColor: 'var(--bg-card)', 
                  padding: '10px', 
                  border: '1px solid var(--border-color)',
                  borderRadius: '5px',
                  boxShadow: '0 2px 10px rgba(0,0,0,0.5)'
              }}>
                  <p className="label mb-1 fw-bold">{data.fullTime}</p>
                  <p className="mb-1" style={{ color: data.status === 'up' ? 'var(--status-up)' : data.status === 'down' ? 'var(--status-down)' : 'var(--status-pending)' }}>
                      {t('history.status')}: {data.status}
                  </p>
                  <p className="mb-1">{t('history.latency')}: {data.latency}ms</p>
                  {data.status === 'down' && (
                      <p className="text-danger small mb-0" style={{maxWidth: '300px', whiteSpace: 'pre-wrap', borderTop: '1px solid #444', marginTop: '5px', paddingTop: '5px'}}>
                          {data.message}
                      </p>
                  )}
              </div>
          );
      }
      return null;
  };

  const CustomizedDot = (props: any) => {
      const { cx, cy, payload } = props;
      if (payload.status === 'down') {
          return (
              <circle cx={cx} cy={cy} r={4} stroke="none" fill="var(--status-down)" />
          );
      }
      return (
          <circle cx={cx} cy={cy} r={3} stroke="none" fill="var(--status-up)" />
      );
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
            <Button 
                variant={monitor.isPaused ? "outline-success" : "outline-warning"} 
                size="sm" 
                onClick={handlePauseToggle}
            >
                {monitor.isPaused ? <FaPlay className="me-2" /> : <FaPause className="me-2" />} 
                {monitor.isPaused ? t('monitor.resume', {defaultValue: 'Resume'}) : t('monitor.pause', {defaultValue: 'Pause'})}
            </Button>
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

      {monitor.type === 'container' && (
        <div className="kuba-card mb-4">
            <div className="kuba-card-header">
                <span><FaBox className="me-2" /> {t('monitor.container')}</span>
                {linkedContainer ? (
                    <Badge bg={linkedContainer.state === 'running' ? 'success' : 'danger'}>{linkedContainer.state}</Badge>
                ) : (
                    <Badge bg="secondary">Unknown</Badge>
                )}
            </div>
            
            {!linkedContainer && (
                <Alert variant="warning">
                    {t('monitor.containerNotFound')}
                    <br />
                    <small className="text-muted">Target: {monitor.containerName || monitor.container?.containerId || 'N/A'}</small>
                </Alert>
            )}

            <div className="d-flex flex-wrap gap-3 align-items-center mb-4">
                <ButtonGroup>
                    <Button 
                        variant="success" 
                        onClick={() => handleAction('start')} 
                        disabled={actionLoading || linkedContainer?.state === 'running'}
                    >
                        <FaPlay className="me-2" /> {t('monitor.start')}
                    </Button>
                    <Button 
                        variant="danger" 
                        onClick={() => handleAction('stop')} 
                        disabled={actionLoading || (linkedContainer && linkedContainer.state !== 'running')}
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
                    <span className="me-2 text-secondary">{t('monitor.restartPolicy')}:</span>
                    <Form.Select 
                        size="sm" 
                        style={{width: 'auto'}} 
                        value={targetPolicy}
                        onChange={(e) => handleRestartPolicy(e.target.value)}
                        className="bg-card text-primary border-secondary"
                    >
                        <option value="no">{t('monitor.restartPolicies.no')}</option>
                        <option value="always">{t('monitor.restartPolicies.always')}</option>
                        <option value="on-failure">{t('monitor.restartPolicies.on-failure')}</option>
                        <option value="unless-stopped">{t('monitor.restartPolicies.unless-stopped')}</option>
                    </Form.Select>
                </div>
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
                      <div>{monitor.lastCheck ? new Date(monitor.lastCheck).toLocaleString() : '-'}</div>
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

      <div className="kuba-card mt-4">
          <div className="kuba-card-header">{t('monitor.history', { defaultValue: 'History' })}</div>
          
          {history.length > 0 && (
              <div style={{ width: '100%', height: 300, marginBottom: '20px' }}>
                  <ResponsiveContainer>
                      <LineChart
                          data={chartData}
                          margin={{
                              top: 5,
                              right: 30,
                              left: 20,
                              bottom: 5,
                          }}
                      >
                          <defs>
                            <linearGradient id={`colorStatus-${monitor.id}`} x1="0" y1="0" x2="1" y2="0">
                              {chartData.map((entry, index) => (
                                <stop 
                                  key={index} 
                                  offset={`${(chartData.length > 1 ? index / (chartData.length - 1) : 0) * 100}%`} 
                                  stopColor={entry.status === 'up' ? 'var(--status-up)' : 'var(--status-down)'} 
                                />
                              ))}
                            </linearGradient>
                          </defs>
                          <CartesianGrid strokeDasharray="3 3" stroke="#444" vertical={false} />
                          <XAxis 
                            dataKey="time" 
                            stroke="var(--text-secondary)" 
                            tick={{fontSize: 12}}
                            minTickGap={30}
                          />
                          <YAxis 
                            stroke="var(--text-secondary)" 
                            tick={{fontSize: 12}}
                            unit="ms"
                            padding={{ top: 20, bottom: 10 }}
                            tickFormatter={(val) => val < 0 ? '' : val}
                            domain={[0, (dataMax: number) => Math.max(dataMax || 0, 10)]}
                          />
                          <Tooltip content={<CustomTooltip />} />
                          <Line 
                            type="monotone" 
                            dataKey="latency" 
                            stroke={`url(#colorStatus-${monitor.id})`} 
                            strokeWidth={3}
                            dot={<CustomizedDot />}
                            activeDot={{ r: 6, fill: 'var(--text-primary)' }}
                            isAnimationActive={false}
                          />
                      </LineChart>
                  </ResponsiveContainer>
              </div>
          )}

          {history.length === 0 && (
              <div className="text-center text-secondary py-4">
                  {t('history.noData', { defaultValue: 'No history data available yet' })}
              </div>
          )}
      </div>

      {monitor.type === 'container' && (
        <div className="kuba-card mt-4">
            <div className="kuba-card-header">
                <span><FaTerminal className="me-2" /> {t('detail.logs')}</span>
            </div>
            <div className="log-viewer mb-3">
                {loadingLogs ? (
                    <div className="text-center p-4">
                        <Spinner animation="border" variant="light" />
                    </div>
                ) : (
                    <pre className="m-0" style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                        {logs || (containerId ? t('monitor.noLogs') : t('monitor.containerNotAttached', { defaultValue: 'Container not attached' }))}
                    </pre>
                )}
                <div ref={logsEndRef} />
            </div>
            
            <div className="d-flex justify-content-end">
                <Button variant="outline-light" size="sm" onClick={fetchLogs} disabled={loadingLogs || !containerId}>
                     <FaTerminal className="me-2" /> {t('monitor.refreshLogs')}
                </Button>
            </div>
        </div>
      )}

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
