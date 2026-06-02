/**
 * Security Workbench（安全 WAF 模块主页面，任务 18.1）。
 *
 * 实现 4 个主要域标签页：规则源 (Source) / 策略中心 (Policy) / 监控观测 (Observe) / 发布运维 (Ops)。
 * 子路由页面通过 defaultTab 属性指定默认激活的标签页。
 */
import { useState, useCallback, useEffect } from 'react';
import {
  Tabs, Table, Button, Modal, Form, Input, Select, Switch, Space, Card,
  Tag, message, Spin, Popconfirm, Descriptions, Row, Col, Statistic,
} from 'antd';
import {
  PlusOutlined, SyncOutlined, CheckCircleOutlined,
  PlayCircleOutlined, RollbackOutlined, EyeOutlined,
  DeleteOutlined, EditOutlined, ReloadOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import {
  fetchWafSourceList, createWafSource, updateWafSource, deleteWafSource,
  checkWafSource, syncWafSource, fetchWafEngineStatus,
  type WafSourceItem, type WafEngineStatusResp,
} from '@/services/caddy-source';
import {
  fetchWafPolicyList, createWafPolicy, updateWafPolicy, deleteWafPolicy,
  publishWafPolicy, previewWafPolicy, rollbackWafPolicy,
  fetchWafPolicyRevisionList,
  type WafPolicyItem, type WafPolicyRevisionItem,
} from '@/services/caddy-policy';
import {
  fetchWafPolicyStats, fetchWafPolicyFalsePositiveFeedbackList,
  type WafPolicyStatsResp, type WafPolicyFalsePositiveFeedbackItem,
} from '@/services/caddy-observe';
import {
  fetchWafReleaseList, activateWafRelease, rollbackWafRelease,
  fetchWafJobList,
  type WafReleaseItem, type WafJobItem,
} from '@/services/caddy-release-job';

// ──────────────────────────────────────────────────────────────────────────
// Source Tab
// ──────────────────────────────────────────────────────────────────────────

function SourceTab() {
  const intl = useIntl();
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
      if (!error) { message.success('更新成功'); setModalOpen(false); loadData(); }
      else message.error('更新失败');
    } else {
      const { error } = await createWafSource(values);
      if (!error) { message.success('创建成功'); setModalOpen(false); loadData(); }
      else message.error('创建失败');
    }
  };

  const handleDelete = async (id: number) => {
    const { error } = await deleteWafSource(id);
    if (!error) { message.success('删除成功'); loadData(); }
    else message.error('删除失败');
  };

  const handleCheck = async (id: number) => {
    setActionLoading(prev => ({ ...prev, [id]: true }));
    const { error } = await checkWafSource(id);
    setActionLoading(prev => ({ ...prev, [id]: false }));
    if (!error) { message.success('检查完成'); loadData(); }
    else message.error('检查失败');
  };

  const handleSync = async (id: number) => {
    setActionLoading(prev => ({ ...prev, [id]: true }));
    const { error } = await syncWafSource(id, false);
    setActionLoading(prev => ({ ...prev, [id]: false }));
    if (!error) { message.success('同步完成'); loadData(); }
    else message.error('同步失败');
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', width: 150 },
    { title: '类型', dataIndex: 'kind', width: 80, render: (v: string) => <Tag>{v}</Tag> },
    { title: '模式', dataIndex: 'mode', width: 80 },
    { title: 'URL', dataIndex: 'url', ellipsis: true },
    { title: '启用', dataIndex: 'enabled', width: 60, render: (v: boolean) => v ? '✓' : '✗' },
    { title: '最后版本', dataIndex: 'lastRelease', width: 100 },
    {
      title: '操作', width: 260, render: (_: unknown, record: WafSourceItem) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>编辑</Button>
          <Button size="small" icon={<CheckCircleOutlined />} loading={actionLoading[record.id]} onClick={() => handleCheck(record.id)}>检查</Button>
          <Button size="small" icon={<SyncOutlined />} loading={actionLoading[record.id]} onClick={() => handleSync(record.id)}>同步</Button>
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
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
            <Col span={6}><Statistic title="当前版本" value={engineStatus.currentVersion || 'N/A'} /></Col>
            <Col span={6}><Statistic title="最新版本" value={engineStatus.latestVersion || 'N/A'} /></Col>
            <Col span={6}><Statistic title="可升级" value={engineStatus.canUpgrade ? '是' : '否'} /></Col>
            <Col span={6}><Statistic title="来源" value={engineStatus.source || 'N/A'} /></Col>
          </Row>
        </Card>
      )}
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>新增规则源</Button>
        <Button icon={<ReloadOutlined />} style={{ marginLeft: 8 }} onClick={loadData}>刷新</Button>
      </div>
      <Table
        dataSource={data} rowKey="id" columns={columns} loading={loading}
        pagination={{ current: page, pageSize, total, onChange: (p, ps) => { setPage(p); setPageSize(ps); } }}
      />
      <Modal title={editingItem ? '编辑规则源' : '新增规则源'} open={modalOpen} onOk={handleSave} onCancel={() => setModalOpen(false)} width={600}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="kind" label="类型" initialValue="crs">
            <Select options={[{ label: 'CRS', value: 'crs' }, { label: 'Coraza Engine', value: 'coraza_engine' }]} />
          </Form.Item>
          <Form.Item name="mode" label="模式" initialValue="remote">
            <Select options={[{ label: '远程', value: 'remote' }, { label: '手动', value: 'manual' }]} />
          </Form.Item>
          <Form.Item name="url" label="URL"><Input /></Form.Item>
          <Form.Item name="checksumUrl" label="校验 URL"><Input /></Form.Item>
          <Form.Item name="schedule" label="调度"><Input placeholder="0 3 * * *" /></Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item>
        </Form>
      </Modal>
    </>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Policy Tab
// ──────────────────────────────────────────────────────────────────────────

function PolicyTab() {
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
      if (!error) { message.success('更新成功'); setModalOpen(false); loadData(); }
    } else {
      const { error } = await createWafPolicy(values);
      if (!error) { message.success('创建成功'); setModalOpen(false); loadData(); }
    }
  };

  const handleDelete = async (id: number) => {
    const { error } = await deleteWafPolicy(id);
    if (!error) { message.success('删除成功'); loadData(); }
  };

  const handlePublish = async (id: number) => {
    const { error } = await publishWafPolicy(id);
    if (!error) message.success('发布成功');
    else message.error('发布失败');
    loadData();
  };

  const handlePreview = async (id: number) => {
    const { data: resp } = await previewWafPolicy(id);
    if (resp) { setPreviewContent(resp.directives || ''); setPreviewOpen(true); }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', width: 150 },
    { title: '引擎模式', dataIndex: 'engineMode', width: 100, render: (v: string) => <Tag>{v}</Tag> },
    { title: '启用', dataIndex: 'enabled', width: 60, render: (v: boolean) => v ? '✓' : '✗' },
    { title: '默认', dataIndex: 'isDefault', width: 60, render: (v: boolean) => v ? '✓' : '✗' },
    { title: 'CRS模板', dataIndex: 'crsTemplate', width: 100 },
    {
      title: '操作', width: 280, render: (_: unknown, record: WafPolicyItem) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => { setEditingItem(record); form.setFieldsValue(record); setModalOpen(true); }}>编辑</Button>
          <Button size="small" icon={<EyeOutlined />} onClick={() => handlePreview(record.id)}>预览</Button>
          <Button size="small" icon={<PlayCircleOutlined />} onClick={() => handlePublish(record.id)}>发布</Button>
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditingItem(null); form.resetFields(); setModalOpen(true); }}>新增策略</Button>
        <Button icon={<ReloadOutlined />} style={{ marginLeft: 8 }} onClick={loadData}>刷新</Button>
      </div>
      <Table
        dataSource={data} rowKey="id" columns={columns} loading={loading}
        pagination={{ current: page, pageSize, total, onChange: (p, ps) => { setPage(p); setPageSize(ps); } }}
      />
      <Modal title={editingItem ? '编辑策略' : '新增策略'} open={modalOpen} onOk={handleSave} onCancel={() => setModalOpen(false)} width={700}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea rows={2} /></Form.Item>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="engineMode" label="引擎模式" initialValue="on">
                <Select options={[{ label: 'On', value: 'on' }, { label: 'Off', value: 'off' }, { label: 'Detection Only', value: 'detectiononly' }]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="crsTemplate" label="CRS 模板" initialValue="balanced">
                <Select options={[{ label: 'Low FP', value: 'low_fp' }, { label: 'Balanced', value: 'balanced' }, { label: 'High Blocking', value: 'high_blocking' }, { label: 'Custom', value: 'custom' }]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item>
            </Col>
          </Row>
          <Form.Item name="config" label="配置（WAF 指令）"><Input.TextArea rows={6} placeholder="SecRuleEngine On..." /></Form.Item>
        </Form>
      </Modal>
      <Modal title="策略预览" open={previewOpen} onCancel={() => setPreviewOpen(false)} footer={null} width={800}>
        <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 4, maxHeight: 500, overflow: 'auto' }}>{previewContent}</pre>
      </Modal>
    </>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Observe Tab
// ──────────────────────────────────────────────────────────────────────────

function ObserveTab() {
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<WafPolicyStatsResp | null>(null);
  const [feedbackList, setFeedbackList] = useState<WafPolicyFalsePositiveFeedbackItem[]>([]);
  const [fbTotal, setFbTotal] = useState(0);
  const [fbPage, setFbPage] = useState(1);

  const loadStats = useCallback(async () => {
    setLoading(true);
    try {
      const { data: resp } = await fetchWafPolicyStats({});
      if (resp) setStats(resp);
    } finally { setLoading(false); }
  }, []);

  const loadFeedback = useCallback(async () => {
    const { data: resp } = await fetchWafPolicyFalsePositiveFeedbackList({
      page: fbPage, pageSize: 10,
    });
    if (resp) { setFeedbackList(resp.list || []); setFbTotal(resp.total || 0); }
  }, [fbPage]);

  useEffect(() => { loadStats(); loadFeedback(); }, [loadStats, loadFeedback]);

  const fbColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '策略', dataIndex: 'policyName', width: 120 },
    { title: 'Host', dataIndex: 'host', width: 120 },
    { title: 'Path', dataIndex: 'path', ellipsis: true },
    { title: 'Method', dataIndex: 'method', width: 80 },
    { title: '状态', dataIndex: 'feedbackStatus', width: 100, render: (v: string) => <Tag color={v === 'resolved' ? 'green' : v === 'confirmed' ? 'blue' : 'orange'}>{v}</Tag> },
    { title: '原因', dataIndex: 'reason', ellipsis: true },
    { title: '创建时间', dataIndex: 'createdAt', width: 170 },
  ];

  return (
    <Spin spinning={loading}>
      {stats?.summary && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col span={6}><Statistic title="命中总数" value={stats.summary.hitCount} /></Col>
            <Col span={6}><Statistic title="拦截数" value={stats.summary.blockedCount} /></Col>
            <Col span={6}><Statistic title="放行数" value={stats.summary.allowedCount} /></Col>
            <Col span={6}><Statistic title="疑似误报" value={stats.summary.suspectedFalsePositiveCount} /></Col>
          </Row>
        </Card>
      )}
      <Card title="误报反馈" size="small">
        <Table
          dataSource={feedbackList} rowKey="id" columns={fbColumns} size="small"
          pagination={{ current: fbPage, pageSize: 10, total: fbTotal, onChange: setFbPage }}
        />
      </Card>
    </Spin>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Ops Tab
// ──────────────────────────────────────────────────────────────────────────

function OpsTab() {
  const [releaseLoading, setReleaseLoading] = useState(false);
  const [releases, setReleases] = useState<WafReleaseItem[]>([]);
  const [relTotal, setRelTotal] = useState(0);
  const [jobLoading, setJobLoading] = useState(false);
  const [jobs, setJobs] = useState<WafJobItem[]>([]);
  const [jobTotal, setJobTotal] = useState(0);

  const loadReleases = useCallback(async () => {
    setReleaseLoading(true);
    try {
      const { data: resp } = await fetchWafReleaseList({ page: 1, pageSize: 10, kind: '', status: '' });
      if (resp) { setReleases(resp.list || []); setRelTotal(resp.total || 0); }
    } finally { setReleaseLoading(false); }
  }, []);

  const loadJobs = useCallback(async () => {
    setJobLoading(true);
    try {
      const { data: resp } = await fetchWafJobList({ page: 1, pageSize: 10, status: '', action: '' });
      if (resp) { setJobs(resp.list || []); setJobTotal(resp.total || 0); }
    } finally { setJobLoading(false); }
  }, []);

  useEffect(() => { loadReleases(); loadJobs(); }, [loadReleases, loadJobs]);

  const handleActivate = async (id: number) => {
    const { error } = await activateWafRelease(id);
    if (!error) { message.success('激活成功'); loadReleases(); }
    else message.error('激活失败');
  };

  const handleRollback = () => {
    Modal.confirm({
      title: '确认回滚？',
      content: '将回滚到上一个稳定版本',
      onOk: async () => {
        const { error } = await rollbackWafRelease({ target: 'last_good' });
        if (!error) { message.success('回滚成功'); loadReleases(); }
        else message.error('回滚失败');
      },
    });
  };

  const relColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '类型', dataIndex: 'kind', width: 80, render: (v: string) => <Tag>{v}</Tag> },
    { title: '版本', dataIndex: 'version', width: 120 },
    { title: '状态', dataIndex: 'status', width: 100, render: (v: string) => <Tag color={v === 'active' ? 'green' : v === 'failed' ? 'red' : 'default'}>{v}</Tag> },
    { title: '大小', dataIndex: 'sizeBytes', width: 100, render: (v: number) => v ? `${(v / 1024).toFixed(1)} KB` : '-' },
    { title: '创建时间', dataIndex: 'createdAt', width: 170 },
    {
      title: '操作', width: 160, render: (_: unknown, record: WafReleaseItem) => (
        <Space>
          <Button size="small" icon={<PlayCircleOutlined />} onClick={() => handleActivate(record.id)} disabled={record.status === 'active'}>激活</Button>
          <Button size="small" icon={<RollbackOutlined />} onClick={handleRollback}>回滚</Button>
        </Space>
      ),
    },
  ];

  const jobColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '操作', dataIndex: 'action', width: 100 },
    { title: '触发', dataIndex: 'triggerMode', width: 80 },
    { title: '操作人', dataIndex: 'operator', width: 100 },
    { title: '状态', dataIndex: 'status', width: 100, render: (v: string) => <Tag color={v === 'success' ? 'green' : v === 'failed' ? 'red' : 'blue'}>{v}</Tag> },
    { title: '消息', dataIndex: 'message', ellipsis: true },
    { title: '开始时间', dataIndex: 'startedAt', width: 170 },
    { title: '结束时间', dataIndex: 'finishedAt', width: 170 },
  ];

  return (
    <>
      <Card title="发布管理" size="small" style={{ marginBottom: 16 }}>
        <Table dataSource={releases} rowKey="id" columns={relColumns} loading={releaseLoading} size="small"
          pagination={{ total: relTotal, pageSize: 10 }} />
      </Card>
      <Card title="任务审计" size="small">
        <Table dataSource={jobs} rowKey="id" columns={jobColumns} loading={jobLoading} size="small"
          pagination={{ total: jobTotal, pageSize: 10 }} />
      </Card>
    </>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Main Workbench
// ──────────────────────────────────────────────────────────────────────────

interface SecurityWorkbenchProps {
  defaultTab?: string;
}

export default function SecurityWorkbench({ defaultTab = 'source' }: SecurityWorkbenchProps) {
  const intl = useIntl();
  const [activeTab, setActiveTab] = useState(defaultTab);

  const tabItems = [
    { key: 'source', label: intl.formatMessage({ id: 'route.security_source', defaultMessage: '规则源' }), children: <SourceTab /> },
    { key: 'policy', label: intl.formatMessage({ id: 'route.security_policy', defaultMessage: '策略中心' }), children: <PolicyTab /> },
    { key: 'observe', label: intl.formatMessage({ id: 'route.security_observe', defaultMessage: '监控观测' }), children: <ObserveTab /> },
    { key: 'ops', label: intl.formatMessage({ id: 'route.security_ops', defaultMessage: '发布运维' }), children: <OpsTab /> },
  ];

  return (
    <PageContainer title={intl.formatMessage({ id: 'route.security', defaultMessage: '安全 WAF' })}>
      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} size="large" />
    </PageContainer>
  );
}
