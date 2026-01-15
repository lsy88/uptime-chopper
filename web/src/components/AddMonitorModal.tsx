import React, { useState, useEffect } from 'react';
import { Modal, Button, Form, Alert, Row, Col, InputGroup, Badge } from 'react-bootstrap';
import { MonitorConfig, MonitorType, Container, getContainers, createMonitor, updateMonitor, getNotifications, NotificationWebhook, createNotification, deleteNotification } from '../api';
import { useTranslation } from 'react-i18next';
import { FaPlus, FaTrash, FaCheck, FaTimes } from 'react-icons/fa';

interface AddMonitorModalProps {
    show: boolean;
    onHide: () => void;
    onSaved: () => void;
    initialData?: MonitorConfig;
}

const AddMonitorModal: React.FC<AddMonitorModalProps> = ({ show, onHide, onSaved, initialData }) => {
    const { t } = useTranslation();
    const [name, setName] = useState('');
    const [type, setType] = useState<MonitorType>('http');
    const [url, setUrl] = useState('');
    const [containerId, setContainerId] = useState('');
    const [interval, setInterval] = useState<number | string>(60);
    const [timeout, setTimeout] = useState<number | string>(10);
    const [containers, setContainers] = useState<Container[]>([]);
    const [notifications, setNotifications] = useState<NotificationWebhook[]>([]);
    const [selectedNotifications, setSelectedNotifications] = useState<string[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);

    // New Notification State
    const [showAddNotif, setShowAddNotif] = useState(false);
    const [newNotifName, setNewNotifName] = useState('');
    const [newNotifType, setNewNotifType] = useState('discord');
    const [newNotifUrl, setNewNotifUrl] = useState('');
    const [addingNotif, setAddingNotif] = useState(false);

    // Container specific options
    const [autoStart, setAutoStart] = useState(false); // If down, start it
    const [autoRestart, setAutoRestart] = useState(false); // If down, restart it
    
    useEffect(() => {
        if (show) {
            loadNotifications();
            if (initialData) {
                setName(initialData.name);
                setType(initialData.type);
                setInterval(initialData.intervalSeconds);
                setTimeout(initialData.timeoutSeconds);
                setSelectedNotifications(initialData.notifyWebhookIds || []);
                if (initialData.type === 'http' && initialData.http) {
                    setUrl(initialData.http.url);
                }
                if (initialData.type === 'container' && initialData.container) {
                    setContainerId(initialData.container.containerId);
                    const action = initialData.container.remediation?.action;
                    setAutoStart(action === 'start');
                    setAutoRestart(action === 'restart');
                }
            } else {
                resetForm();
            }
            if (type === 'container' || (initialData && initialData.type === 'container')) {
                loadContainers();
            }
        }
    }, [show, initialData]);

    const resetForm = () => {
        setName('');
        setType('http');
        setUrl('');
        setContainerId('');
        setInterval(60);
        setTimeout(10);
        setAutoStart(false);
        setAutoRestart(false);
        setSelectedNotifications([]);
        setError(null);
    };

    const loadNotifications = async () => {
        try {
            const data = await getNotifications();
            setNotifications(data || []);
        } catch (err) {
            console.error("Failed to load notifications", err);
        }
    };

    const loadContainers = async () => {
        try {
            const data = await getContainers();
            setContainers(data);
            if (data.length > 0 && !containerId) {
                setContainerId(data[0].id);
            }
        } catch (err) {
            console.error(err);
            setError("Failed to load containers");
        }
    };

    const toggleNotification = (id: string) => {
        if (selectedNotifications.includes(id)) {
            setSelectedNotifications(selectedNotifications.filter(n => n !== id));
        } else {
            setSelectedNotifications([...selectedNotifications, id]);
        }
    };

    const handleAddNotification = async () => {
        if (!newNotifName || !newNotifUrl) return;
        setAddingNotif(true);
        try {
            await createNotification({
                name: newNotifName,
                type: newNotifType,
                url: newNotifUrl
            });
            await loadNotifications();
            setNewNotifName('');
            setNewNotifUrl('');
            setNewNotifType('discord');
            setShowAddNotif(false);
        } catch (err: any) {
            console.error(err);
            setError("Failed to add notification: " + err.message);
        } finally {
            setAddingNotif(false);
        }
    };

    const handleDeleteNotification = async (id: string, e: React.MouseEvent) => {
        e.stopPropagation();
        if (!window.confirm("Delete this notification channel?")) return;
        try {
            await deleteNotification(id);
            await loadNotifications();
            // Also unselect it if selected
            if (selectedNotifications.includes(id)) {
                setSelectedNotifications(selectedNotifications.filter(n => n !== id));
            }
        } catch (err: any) {
             setError("Failed to delete notification: " + err.message);
        }
    };

    const handleSubmit = async () => {
        setError(null);
        if (!name) {
            setError("Name is required");
            return;
        }

        if (type === 'http' && !url) {
            setError("URL is required");
            return;
        }

        if (type === 'container' && !containerId) {
            setError("Container ID is required");
            return;
        }

        setSaving(true);
        try {
            const payload: any = {
                name,
                type,
                intervalSeconds: Number(interval),
                timeoutSeconds: Number(timeout),
                notifyWebhookIds: selectedNotifications,
                logs: { include: false, tail: 50 } // Default
            };

            if (type === 'http') {
                payload.http = { url };
            } else if (type === 'container') {
                let action: "none" | "start" | "restart" = "none";
                if (autoRestart) action = "restart";
                else if (autoStart) action = "start";

                payload.container = {
                    containerId,
                    remediation: {
                        action: action,
                        maxAttempts: 3,
                        cooldownSeconds: 60
                    }
                };
                payload.logs.include = true; // Enable logs by default for containers
            }

            await (initialData ? updateMonitor(initialData.id, payload) : createMonitor(payload));
            onSaved();
            onHide();
            resetForm();
        } catch (err: any) {
            setError(err.message || "Failed to save monitor");
        } finally {
            setSaving(false);
        }
    };

    return (
        <Modal show={show} onHide={onHide} centered size="lg" contentClassName="bg-card text-primary border-secondary">
            <Modal.Header closeButton className="border-secondary">
                <Modal.Title>{initialData ? t('editMonitor') : t('addMonitor')}</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {error && <Alert variant="danger">{error}</Alert>}
                
                <Form>
                    <Form.Group className="mb-3">
                        <Form.Label>{t('monitor.type')}</Form.Label>
                        <Form.Select 
                            value={type} 
                            onChange={(e) => {
                                const newType = e.target.value as MonitorType;
                                setType(newType);
                                if (newType === 'container') {
                                    loadContainers();
                                }
                            }}
                            className="bg-body text-primary border-secondary"
                        >
                            <option value="http">{t('monitor.types.http')}</option>
                            <option value="container">{t('monitor.types.container')}</option>
                        </Form.Select>
                    </Form.Group>

                    <Form.Group className="mb-3">
                        <Form.Label>{t('monitor.name')}</Form.Label>
                        <Form.Control 
                            type="text" 
                            placeholder="My Website" 
                            value={name}
                            onChange={e => setName(e.target.value)}
                            className="bg-body text-primary border-secondary"
                        />
                    </Form.Group>

                    {type === 'http' && (
                        <Form.Group className="mb-3">
                            <Form.Label>{t('monitor.url')}</Form.Label>
                            <Form.Control 
                                type="text" 
                                placeholder="https://example.com" 
                                value={url}
                                onChange={e => setUrl(e.target.value)}
                                className="bg-body text-primary border-secondary"
                            />
                        </Form.Group>
                    )}

                    {type === 'container' && (
                        <>
                            <Form.Group className="mb-3">
                                <Form.Label>{t('monitor.container')}</Form.Label>
                                <Form.Select 
                                    value={containerId} 
                                    onChange={e => setContainerId(e.target.value)}
                                    className="bg-body text-primary border-secondary"
                                >
                                    <option value="">Select a container...</option>
                                    {containers.map(c => (
                                        <option key={c.id} value={c.id}>
                                            {c.names && c.names.length > 0 ? c.names[0].replace('/', '') : c.id.substring(0, 12)} - {c.image}
                                        </option>
                                    ))}
                                </Form.Select>
                            </Form.Group>
                            
                            <Row className="mb-3">
                                <Col>
                                    <Form.Check 
                                        type="switch"
                                        id="auto-start-switch"
                                        label={t('monitor.autoRestart')}
                                        checked={autoRestart}
                                        onChange={e => {
                                            setAutoRestart(e.target.checked);
                                            if (e.target.checked) setAutoStart(false);
                                        }}
                                        className="text-secondary"
                                    />
                                    <Form.Text className="text-muted">
                                        If the container stops unexpectedly, try to restart it.
                                    </Form.Text>
                                </Col>
                            </Row>
                        </>
                    )}

                    <Form.Group className="mb-3">
                        <div className="mb-2">
                            <Form.Label className="mb-0">{t('monitor.notifications')}</Form.Label>
                        </div>

                        {showAddNotif && (
                            <div className="card p-3 mb-3 border-secondary bg-body text-primary">
                                <h6 className="mb-3">{t('monitor.addNotification', {defaultValue: 'Add Notification'})}</h6>
                                <Row className="g-2 mb-2">
                                    <Col md={4}>
                                        <Form.Control 
                                            placeholder={t('monitor.notificationName', {defaultValue: 'Name'})}
                                            value={newNotifName}
                                            onChange={e => setNewNotifName(e.target.value)}
                                            size="sm"
                                            className="bg-body text-primary border-secondary"
                                        />
                                    </Col>
                                    <Col md={3}>
                                        <Form.Select 
                                            value={newNotifType} 
                                            onChange={e => setNewNotifType(e.target.value)}
                                            size="sm"
                                            className="bg-body text-primary border-secondary"
                                        >
                                            <option value="discord">{t('monitor.notificationTypes.discord', {defaultValue: 'Discord'})}</option>
                                            <option value="dingtalk">{t('monitor.notificationTypes.dingtalk', {defaultValue: 'DingTalk'})}</option>
                                            <option value="wechat">{t('monitor.notificationTypes.wechat', {defaultValue: 'WeChat'})}</option>
                                        </Form.Select>
                                    </Col>
                                    <Col md={5}>
                                        <Form.Control 
                                            placeholder={t('monitor.notificationUrl', {defaultValue: 'Webhook URL'})}
                                            value={newNotifUrl}
                                            onChange={e => setNewNotifUrl(e.target.value)}
                                            size="sm"
                                            className="bg-body text-primary border-secondary"
                                        />
                                    </Col>
                                </Row>
                                <div className="text-end">
                                    <Button variant="link" size="sm" className="text-secondary me-2 text-decoration-none" onClick={() => setShowAddNotif(false)}>
                                        {t('cancel')}
                                    </Button>
                                    <Button variant="primary" size="sm" onClick={handleAddNotification} disabled={addingNotif}>
                                        {addingNotif ? 'Saving...' : t('save')}
                                    </Button>
                                </div>
                            </div>
                        )}

                        {notifications.length === 0 ? (
                            <div 
                                className="text-secondary small text-decoration-underline" 
                                style={{ cursor: 'pointer' }}
                                onClick={() => setShowAddNotif(true)}
                            >
                                {t('monitor.noNotifications')}
                            </div>
                        ) : (
                            <div className="d-flex flex-wrap gap-2">
                                {notifications.map(n => {
                                    const id = n.id || n.name;
                                    const isSelected = selectedNotifications.includes(id);
                                    return (
                                        <div 
                                            key={id} 
                                            className={`d-flex align-items-center border rounded px-2 py-1 user-select-none ${isSelected ? 'border-primary bg-primary-subtle' : ''}`}
                                            style={{
                                                cursor: 'pointer',
                                                borderColor: isSelected ? undefined : 'var(--border-color)'
                                            }}
                                            onClick={() => toggleNotification(id)}
                                        >
                                            <div 
                                                className={`me-2 d-flex align-items-center justify-content-center rounded-circle border ${isSelected ? 'bg-primary border-primary text-white' : 'border-secondary'}`} 
                                                style={{
                                                    width: 16, 
                                                    height: 16,
                                                    backgroundColor: isSelected ? undefined : 'var(--bg-body)'
                                                }}
                                            >
                                                {isSelected && <FaCheck size={10} />}
                                            </div>
                                            <span className="me-2">{n.name}</span>
                                            <Badge bg="secondary" className="me-2 opacity-75" style={{fontSize: '0.6em'}}>{n.type}</Badge>
                                            {n.editable !== false && (
                                                <FaTimes 
                                                    className="text-danger opacity-50 hover-opacity-100" 
                                                    style={{cursor: 'pointer'}}
                                                    onClick={(e) => handleDeleteNotification(id, e)}
                                                    title={t('monitor.delete')}
                                                />
                                            )}
                                        </div>
                                    );
                                })}
                                <div 
                                    className="d-flex align-items-center border border-secondary rounded px-2 py-1 user-select-none text-secondary"
                                    style={{ cursor: 'pointer', borderStyle: 'dashed' }}
                                    onClick={() => setShowAddNotif(true)}
                                    title={t('monitor.addNotification', {defaultValue: 'Add Notification'})}
                                >
                                    <FaPlus size={12} />
                                </div>
                            </div>
                        )}
                    </Form.Group>

                    <Row>
                        <Col>
                            <Form.Group className="mb-3">
                                <Form.Label>{t('monitor.interval')}</Form.Label>
                                <Form.Control 
                                    type="number" 
                                    value={interval}
                                    onChange={e => setInterval(e.target.value === '' ? '' : Number(e.target.value))}
                                    className="bg-body text-primary border-secondary"
                                />
                            </Form.Group>
                        </Col>
                        <Col>
                            <Form.Group className="mb-3">
                                <Form.Label>{t('monitor.timeout')}</Form.Label>
                                <Form.Control 
                                    type="number" 
                                    value={timeout}
                                    onChange={e => setTimeout(e.target.value === '' ? '' : Number(e.target.value))}
                                    className="bg-body text-primary border-secondary"
                                />
                            </Form.Group>
                        </Col>
                    </Row>
                </Form>
            </Modal.Body>
            <Modal.Footer className="border-secondary">
                <Button variant="secondary" onClick={onHide} disabled={saving}>
                    {t('cancel')}
                </Button>
                <Button variant="success" onClick={handleSubmit} disabled={saving}>
                    {saving ? 'Saving...' : t('save')}
                </Button>
            </Modal.Footer>
        </Modal>
    );
};

export default AddMonitorModal;