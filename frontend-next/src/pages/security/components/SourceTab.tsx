import { useState, useCallback, useEffect } from 'react';
import {
  Table, Button, Modal, Form, Input, Select, Switch, Space, Card,
  Tag, message, Popconfirm, Descriptions, Row, Col, Statistic,
} from 'antd';
import {
  PlusOutlined, SyncOutlined, CheckCircleOutlined,
  DeleteOutlined, EditOutlined, ReloadOutlined,
} from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  fetchWafSourceList, createWafSource, updateWafSource, deleteWafSource,
  checkWafSource, syncWafSource, fetchWafEngineStatus,
  type WafSourceItem, type WafEngineStatusResp,
} from '@/services/caddy-source';

export default function SourceTab() {
  const intl = useIntl();
  const t = useCallback(
    (id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }),
    [intl],
  );

  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<WafSourceItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<WafSourceItem | null>(null);
  const [form] = Form.useForm();
  const [engineStatus, setEngineStatus] = useState<WafEngineStatusResp | null>(null);
  const [actionLoading, setActionLoading] = useState<Record<number, boolean>>({});

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const { data: resp, error } = await fetchWafSourceList({ page, pageSize, kind: '', name: '' });
      if (!error && resp) {
        setData(resp.list || []);
        setTotal(resp.total || 0);
      }
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  const loadEngineStatus = useCallback(async () => {
    const { data: resp } = await fetchWafEngineStatus();
    if (resp) setEngineStatus(resp);
  }, []);

  useEffect(() => { loadData(); loadEngineStatus(); }, [loadData, loadEngineStatus]);

  const handleAdd = () => { setEditingItem(null); form.resetFields(); setModalOpen(true); };
  const handleEdit = (record: WafSourceItem) => {
    setEditingItem(record); form.setFieldsValue(record); setModalOpen(true);
  };

  const handleSave = async () => {
    const values = await form.validateFields();
    if (editingItem) {
      const { error } = await updateWafSource(editingItem.id, values);
      if (!error) { message.success(t('security.source.updateSuccess', '更新成功')); setModalOpen(false); loadData(); }
      else message.error(t('security.source.updateFailed', '更新失败'));
    } else {
      const { error } = await createWafSource(values);
      if (!error) { message.success(t('security.source.createSuccess', '创建成功')); setModalOpen(false); loadData(); }
      else message.error(t('security.source.createFailed', '创建失败'));
    }
  };

  const handleDelete = async (id: number) => {
    const { error } = await deleteWafSource(id);
    if (!error) { message.success(t('security.source.deleteSuccess', '删除成功')); loadData(); }
    else message.error(t('security.source.deleteFailed', '删除失败'));
  };

  const handleCheck = async (id: number) => {
    setActionLoading(prev => ({ ...prev, [id]: true }));
    const { error } = await checkWafSource(id);
    setActionLoading(prev => ({ ...prev, [id]: false }));
    if (!error) { message.success(t('security.source.checkSuccess', '检查完成')); loadData(); }
    else message.error(t('security.source.checkFailed', '检查失败'));
  };

  const handleSync = async (id: number) => {
    setActionLoading(prev => ({ ...prev, [id]: true }));
    const { error } = await syncWafSource(id, false);
    setActionLoading(prev => ({ ...prev, [id]: false }));
    if (!error) { message.success(t('security.source.syncSuccess', '同步完成')); loadData(); }
    else message.error(t('security.source.syncFailed', '同步失败'));
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: t('security.source.name', '名称'), dataIndex: 'name', width: 150 },
    { title: t('security.source.type', '类型'), dataIndex: 'kind', width: 80, render: (v: string) => <Tag>{v}</Tag> },
    { title: t('security.source.mode', '模式'), dataIndex: 'mode', width: 80 },
    { title: t('security.source.url', 'URL'), dataIndex: 'url', ellipsis: true },
    { title: t('security.source.enabled', '启用'), dataIndex: 'enabled', width: 60, render: (v: boolean) => v ? '✓' : '✗' },
    { title: t('security.source.lastVersion', '最后版本'), dataIndex: 'lastRelease', width: 100 },
    {
      title: t('common.action', '操作'), width: 260, render: (_: unknown, record: WafSourceItem) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>{t('common.edit', '编辑')}</Button>
          <Button size="small" icon={<CheckCircleOutlined />} loading={actionLoading[record.id]} onClick={() => handleCheck(record.id)}>{t('security.source.check', '检查')}</Button>
          <Button size="small" icon={<SyncOutlined />} loading={actionLoading[record.id]} onClick={() => handleSync(record.id)}>{t('security.source.sync', '同步')}</Button>
          <Popconfirm title={t('common.confirmDelete', '确认删除？')} onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />}>{t('common.delete', '删除')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      {engineStatus && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col span={6}><Statistic title={t('security.source.currentVersion', '当前版本')} value={engineStatus.currentVersion || 'N/A'} /></Col>
            <Col span={6}><Statistic title={t('security.source.latestVersion', '最新版本')} value={engineStatus.latestVersion || 'N/A'} /></Col>
            <Col span={6}><Statistic title={t('security.source.upgradable', '可升级')} value={engineStatus.canUpgrade ? t('common.yesOrNo.yes', '是') : t('common.yesOrNo.no', '否')} /></Col>
            <Col span={6}><Statistic title={t('security.source.origin', '来源')} value={engineStatus.source || 'N/A'} /></Col>
          </Row>
        </Card>
      )}
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>{t('security.source.addSource', '新增规则源')}</Button>
        <Button icon={<ReloadOutlined />} style={{ marginLeft: 8 }} onClick={loadData}>{t('common.refresh', '刷新')}</Button>
      </div>
      <Table
        dataSource={data} rowKey="id" columns={columns} loading={loading}
        pagination={{ current: page, pageSize, total, onChange: (p, ps) => { setPage(p); setPageSize(ps); } }}
      />
      <Modal title={editingItem ? t('security.source.editSource', '编辑规则源') : t('security.source.addSource', '新增规则源')} open={modalOpen} onOk={handleSave} onCancel={() => setModalOpen(false)} width={600}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('security.source.name', '名称')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="kind" label={t('security.source.type', '类型')} initialValue="crs">
            <Select options={[{ label: 'CRS', value: 'crs' }, { label: 'Coraza Engine', value: 'coraza_engine' }]} />
          </Form.Item>
          <Form.Item name="mode" label={t('security.source.mode', '模式')} initialValue="remote">
            <Select options={[{ label: t('security.source.remote', '远程'), value: 'remote' }, { label: t('security.source.manual', '手动'), value: 'manual' }]} />
          </Form.Item>
          <Form.Item name="url" label={t('security.source.url', 'URL')}><Input /></Form.Item>
          <Form.Item name="checksumUrl" label={t('security.source.checkUrl', '校验 URL')}><Input /></Form.Item>
          <Form.Item name="schedule" label={t('security.source.schedule', '调度')}><Input placeholder="0 3 * * *" /></Form.Item>
          <Form.Item name="enabled" label={t('security.source.enabled', '启用')} valuePropName="checked"><Switch /></Form.Item>
        </Form>
      </Modal>
    </>
  );
}
