import { useState, useCallback, useEffect } from 'react';
import {
  Table, Button, Modal, Form, Input, Select, Switch, Space,
  Tag, message, Popconfirm, Row, Col,
} from 'antd';
import {
  PlusOutlined, PlayCircleOutlined, EyeOutlined,
  DeleteOutlined, EditOutlined, ReloadOutlined,
} from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  fetchWafPolicyList, createWafPolicy, updateWafPolicy, deleteWafPolicy,
  publishWafPolicy, previewWafPolicy,
  type WafPolicyItem,
} from '@/services/caddy-policy';

export default function PolicyTab() {
  const intl = useIntl();
  const t = useCallback(
    (id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }),
    [intl],
  );

  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<WafPolicyItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<WafPolicyItem | null>(null);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewContent, setPreviewContent] = useState('');
  const [form] = Form.useForm();

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const { data: resp, error } = await fetchWafPolicyList({ page, pageSize, name: '' });
      if (!error && resp) { setData(resp.list || []); setTotal(resp.total || 0); }
    } finally { setLoading(false); }
  }, [page, pageSize]);

  useEffect(() => { loadData(); }, [loadData]);

  const handleSave = async () => {
    const values = await form.validateFields();
    if (editingItem) {
      const { error } = await updateWafPolicy(editingItem.id, values);
      if (!error) { message.success(t('security.policy.updateSuccess', '更新成功')); setModalOpen(false); loadData(); }
    } else {
      const { error } = await createWafPolicy(values);
      if (!error) { message.success(t('security.policy.createSuccess', '创建成功')); setModalOpen(false); loadData(); }
    }
  };

  const handleDelete = async (id: number) => {
    const { error } = await deleteWafPolicy(id);
    if (!error) { message.success(t('security.policy.deleteSuccess', '删除成功')); loadData(); }
  };

  const handlePublish = async (id: number) => {
    const { error } = await publishWafPolicy(id);
    if (!error) message.success(t('security.policy.publishSuccess', '发布成功'));
    else message.error(t('security.policy.publishFailed', '发布失败'));
    loadData();
  };

  const handlePreview = async (id: number) => {
    const { data: resp } = await previewWafPolicy(id);
    if (resp) { setPreviewContent(resp.directives || ''); setPreviewOpen(true); }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: t('security.policy.name', '名称'), dataIndex: 'name', width: 150 },
    { title: t('security.policy.engineMode', '引擎模式'), dataIndex: 'engineMode', width: 100, render: (v: string) => <Tag>{v}</Tag> },
    { title: t('security.policy.enabled', '启用'), dataIndex: 'enabled', width: 60, render: (v: boolean) => v ? '✓' : '✗' },
    { title: t('security.policy.default', '默认'), dataIndex: 'isDefault', width: 60, render: (v: boolean) => v ? '✓' : '✗' },
    { title: t('security.policy.crsTemplate', 'CRS模板'), dataIndex: 'crsTemplate', width: 100 },
    {
      title: t('common.action', '操作'), width: 280, render: (_: unknown, record: WafPolicyItem) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => { setEditingItem(record); form.setFieldsValue(record); setModalOpen(true); }}>{t('common.edit', '编辑')}</Button>
          <Button size="small" icon={<EyeOutlined />} onClick={() => handlePreview(record.id)}>{t('security.policy.preview', '预览')}</Button>
          <Button size="small" icon={<PlayCircleOutlined />} onClick={() => handlePublish(record.id)}>{t('security.policy.publish', '发布')}</Button>
          <Popconfirm title={t('common.confirmDelete', '确认删除？')} onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />}>{t('common.delete', '删除')}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditingItem(null); form.resetFields(); setModalOpen(true); }}>{t('security.policy.addPolicy', '新增策略')}</Button>
        <Button icon={<ReloadOutlined />} style={{ marginLeft: 8 }} onClick={loadData}>{t('common.refresh', '刷新')}</Button>
      </div>
      <Table
        dataSource={data} rowKey="id" columns={columns} loading={loading}
        pagination={{ current: page, pageSize, total, onChange: (p, ps) => { setPage(p); setPageSize(ps); } }}
      />
      <Modal title={editingItem ? t('security.policy.editPolicy', '编辑策略') : t('security.policy.addPolicy', '新增策略')} open={modalOpen} onOk={handleSave} onCancel={() => setModalOpen(false)} width={700}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('security.policy.name', '名称')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label={t('security.policy.description', '描述')}><Input.TextArea rows={2} /></Form.Item>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="engineMode" label={t('security.policy.engineMode', '引擎模式')} initialValue="on">
                <Select options={[{ label: 'On', value: 'on' }, { label: 'Off', value: 'off' }, { label: 'Detection Only', value: 'detectiononly' }]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="crsTemplate" label={t('security.policy.crsTemplate', 'CRS 模板')} initialValue="balanced">
                <Select options={[{ label: 'Low FP', value: 'low_fp' }, { label: 'Balanced', value: 'balanced' }, { label: 'High Blocking', value: 'high_blocking' }, { label: 'Custom', value: 'custom' }]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="enabled" label={t('security.policy.enabled', '启用')} valuePropName="checked"><Switch /></Form.Item>
            </Col>
          </Row>
          <Form.Item name="config" label={t('security.policy.config', '配置（WAF 指令）')}><Input.TextArea rows={6} placeholder="SecRuleEngine On..." /></Form.Item>
        </Form>
      </Modal>
      <Modal title={t('security.policy.previewTitle', '策略预览')} open={previewOpen} onCancel={() => setPreviewOpen(false)} footer={null} width={800}>
        <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 4, maxHeight: 500, overflow: 'auto' }}>{previewContent}</pre>
      </Modal>
    </>
  );
}
