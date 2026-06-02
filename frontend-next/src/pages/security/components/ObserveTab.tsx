import { useState, useCallback, useEffect } from 'react';
import {
  Table, Card, Tag, Spin, Row, Col, Statistic,
} from 'antd';
import { useIntl } from '@umijs/max';
import {
  fetchWafPolicyStats, fetchWafPolicyFalsePositiveFeedbackList,
  type WafPolicyStatsResp, type WafPolicyFalsePositiveFeedbackItem,
} from '@/services/caddy-observe';

export default function ObserveTab() {
  const intl = useIntl();
  const t = useCallback(
    (id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }),
    [intl],
  );

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
    { title: t('security.policy.name', '策略'), dataIndex: 'policyName', width: 120 },
    { title: 'Host', dataIndex: 'host', width: 120 },
    { title: 'Path', dataIndex: 'path', ellipsis: true },
    { title: 'Method', dataIndex: 'method', width: 80 },
    { title: t('security.observe.status', '状态'), dataIndex: 'feedbackStatus', width: 100, render: (v: string) => <Tag color={v === 'resolved' ? 'green' : v === 'confirmed' ? 'blue' : 'orange'}>{v}</Tag> },
    { title: t('security.observe.reason', '原因'), dataIndex: 'reason', ellipsis: true },
    { title: t('common.createdAt', '创建时间'), dataIndex: 'createdAt', width: 170 },
  ];

  return (
    <Spin spinning={loading}>
      {stats?.summary && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col span={6}><Statistic title={t('security.observe.totalHits', '命中总数')} value={stats.summary.hitCount} /></Col>
            <Col span={6}><Statistic title={t('security.observe.blocked', '拦截数')} value={stats.summary.blockedCount} /></Col>
            <Col span={6}><Statistic title={t('security.observe.allowed', '放行数')} value={stats.summary.allowedCount} /></Col>
            <Col span={6}><Statistic title={t('security.observe.falsePositive', '疑似误报')} value={stats.summary.suspectedFalsePositiveCount} /></Col>
          </Row>
        </Card>
      )}
      <Card title={t('security.observe.feedback', '误报反馈')} size="small">
        <Table
          dataSource={feedbackList} rowKey="id" columns={fbColumns} size="small"
          pagination={{ current: fbPage, pageSize: 10, total: fbTotal, onChange: setFbPage }}
        />
      </Card>
    </Spin>
  );
}
