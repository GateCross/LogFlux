import { useState, useCallback, useEffect } from 'react';
import {
  Table, Button, Modal, Space, Card, Tag, message,
} from 'antd';
import {
  PlayCircleOutlined, RollbackOutlined,
} from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import {
  fetchWafReleaseList, activateWafRelease, rollbackWafRelease,
  fetchWafJobList,
  type WafReleaseItem, type WafJobItem,
} from '@/services/caddy-release-job';

export default function OpsTab() {
  const intl = useIntl();
  const t = useCallback(
    (id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }),
    [intl],
  );

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
    if (!error) { message.success(t('security.ops.activateSuccess', '激活成功')); loadReleases(); }
    else message.error(t('security.ops.activateFailed', '激活失败'));
  };

  const handleRollback = () => {
    Modal.confirm({
      title: t('security.ops.rollbackConfirm', '确认回滚？'),
      content: t('security.ops.rollbackDesc', '将回滚到上一个稳定版本'),
      onOk: async () => {
        const { error } = await rollbackWafRelease({ target: 'last_good' });
        if (!error) { message.success(t('security.ops.rollbackSuccess', '回滚成功')); loadReleases(); }
        else message.error(t('security.ops.rollbackFailed', '回滚失败'));
      },
    });
  };

  const relColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: t('security.ops.type', '类型'), dataIndex: 'kind', width: 80, render: (v: string) => <Tag>{v}</Tag> },
    { title: t('security.ops.version', '版本'), dataIndex: 'version', width: 120 },
    { title: t('security.ops.status', '状态'), dataIndex: 'status', width: 100, render: (v: string) => <Tag color={v === 'active' ? 'green' : v === 'failed' ? 'red' : 'default'}>{v}</Tag> },
    { title: t('security.ops.size', '大小'), dataIndex: 'sizeBytes', width: 100, render: (v: number) => v ? `${(v / 1024).toFixed(1)} KB` : '-' },
    { title: t('common.createdAt', '创建时间'), dataIndex: 'createdAt', width: 170 },
    {
      title: t('common.action', '操作'), width: 160, render: (_: unknown, record: WafReleaseItem) => (
        <Space>
          <Button size="small" icon={<PlayCircleOutlined />} onClick={() => handleActivate(record.id)} disabled={record.status === 'active'}>{t('security.ops.activate', '激活')}</Button>
          <Button size="small" icon={<RollbackOutlined />} onClick={handleRollback}>{t('security.ops.rollback', '回滚')}</Button>
        </Space>
      ),
    },
  ];

  const jobColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: t('common.action', '操作'), dataIndex: 'action', width: 100 },
    { title: t('security.ops.trigger', '触发'), dataIndex: 'triggerMode', width: 80 },
    { title: t('security.ops.operator', '操作人'), dataIndex: 'operator', width: 100 },
    { title: t('security.ops.status', '状态'), dataIndex: 'status', width: 100, render: (v: string) => <Tag color={v === 'success' ? 'green' : v === 'failed' ? 'red' : 'blue'}>{v}</Tag> },
    { title: t('security.ops.message', '消息'), dataIndex: 'message', ellipsis: true },
    { title: t('security.ops.startTime', '开始时间'), dataIndex: 'startedAt', width: 170 },
    { title: t('security.ops.endTime', '结束时间'), dataIndex: 'finishedAt', width: 170 },
  ];

  return (
    <>
      <Card title={t('security.ops.releaseManagement', '发布管理')} size="small" style={{ marginBottom: 16 }}>
        <Table dataSource={releases} rowKey="id" columns={relColumns} loading={releaseLoading} size="small"
          pagination={{ total: relTotal, pageSize: 10 }} />
      </Card>
      <Card title={t('security.ops.jobAudit', '任务审计')} size="small">
        <Table dataSource={jobs} rowKey="id" columns={jobColumns} loading={jobLoading} size="small"
          pagination={{ total: jobTotal, pageSize: 10 }} />
      </Card>
    </>
  );
}
