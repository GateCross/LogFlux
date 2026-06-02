/**
 * Dashboard Page (Task 13.1)
 *
 * 统计概览、错误统计、趋势图、地理分布地图、最近日志。
 * 直接作为 ProLayout 的子路由渲染，不需要额外的 PageContainer。
 */
import { useEffect, useState, useCallback } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Spin, Empty, Alert, Space, Select, Typography } from 'antd';
import { useIntl } from '@umijs/max';
import {
  ArrowUpOutlined,
  ArrowDownOutlined,
  EyeOutlined,
  UserOutlined,
  StopOutlined,
  ThunderboltOutlined,
  WarningOutlined,
  BugOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { fetchDashboardSummary } from '@/services/dashboard';
import type { DashboardSummaryResp, DashboardRecentItem } from '@/services/dashboard';
import TrendChart from '@/components/charts/TrendChart';
import MapChart from '@/components/charts/MapChart';

const { Title } = Typography;

const TIME_RANGE_PRESETS: { label: string; value: number }[] = [
  { label: '1 Hour', value: 3600 },
  { label: '6 Hours', value: 21600 },
  { label: '24 Hours', value: 86400 },
  { label: '7 Days', value: 604800 },
  { label: '30 Days', value: 2592000 },
];

export default function DashboardPage() {
  const intl = useIntl();

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<DashboardSummaryResp | null>(null);
  const [rangeSeconds, setRangeSeconds] = useState<number>(86400);

  const loadData = useCallback(async (seconds: number) => {
    setLoading(true);
    setError(null);

    const now = Math.floor(Date.now() / 1000);
    const startTime = new Date((now - seconds) * 1000).toISOString();
    const endTime = new Date(now * 1000).toISOString();

    const { data: result, error: err } = await fetchDashboardSummary({
      startTime,
      endTime,
      intervalSec: Math.floor(seconds / 60),
      recentLimit: 20,
    });

    if (err) {
      setError(err.message || 'Failed to load dashboard data');
      setData(null);
    } else {
      setData(result);
    }

    setLoading(false);
  }, []);

  useEffect(() => {
    loadData(rangeSeconds);
  }, [rangeSeconds, loadData]);

  // --- Recent logs table columns ---
  const columns: ColumnsType<DashboardRecentItem> = [
    {
      title: 'Time',
      dataIndex: 'logTime',
      key: 'logTime',
      width: 180,
      render: (text: string) => (text ? new Date(text).toLocaleString() : '-'),
    },
    {
      title: 'Method',
      dataIndex: 'method',
      key: 'method',
      width: 90,
      render: (method: string) => {
        const colorMap: Record<string, string> = {
          GET: 'blue', POST: 'green', PUT: 'orange', DELETE: 'red', PATCH: 'purple',
        };
        return <Tag color={colorMap[method] || 'default'}>{method}</Tag>;
      },
    },
    { title: 'URI', dataIndex: 'uri', key: 'uri', ellipsis: true },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: number) => {
        let color = 'green';
        if (status >= 400 && status < 500) color = 'orange';
        if (status >= 500) color = 'red';
        return <Tag color={color}>{status}</Tag>;
      },
    },
    { title: 'Remote IP', dataIndex: 'remoteIp', key: 'remoteIp', width: 150 },
    { title: 'Country', dataIndex: 'country', key: 'country', width: 120 },
  ];

  // --- Render ---
  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* Header with time range selector */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={4} style={{ margin: 0 }}>Dashboard</Title>
        <Select
          value={rangeSeconds}
          onChange={(v) => setRangeSeconds(v)}
          style={{ width: 160 }}
          options={TIME_RANGE_PRESETS.map(p => ({ label: p.label, value: p.value }))}
        />
      </div>

      {/* Loading state */}
      {loading && (
        <div style={{ textAlign: 'center', padding: '80px 0' }}>
          <Spin size="large" />
        </div>
      )}

      {/* Error state */}
      {!loading && error && (
        <Alert
          message="Error"
          description={error}
          type="error"
          showIcon
          closable
          action={<a onClick={() => loadData(rangeSeconds)}>Retry</a>}
        />
      )}

      {/* Empty state */}
      {!loading && !error && !data && (
        <Card>
          <Empty description="No dashboard data available" />
        </Card>
      )}

      {/* Data loaded */}
      {!loading && !error && data && (
        <>
          {/* Summary Stats Cards */}
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic title="Requests" value={data.stats.requests} prefix={<ArrowUpOutlined />} valueStyle={{ color: '#1890ff' }} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic title="PV" value={data.stats.pv} prefix={<EyeOutlined />} valueStyle={{ color: '#722ed1' }} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic title="UV" value={data.stats.uv} prefix={<UserOutlined />} valueStyle={{ color: '#13c2c2' }} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic title="Unique IP" value={data.stats.uniqueIp} prefix={<ArrowDownOutlined />} valueStyle={{ color: '#52c41a' }} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic title="Blocked" value={data.stats.blocked} prefix={<StopOutlined />} valueStyle={{ color: '#faad14' }} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic title="Attack IP" value={data.stats.attackIp} prefix={<ThunderboltOutlined />} valueStyle={{ color: '#ff4d4f' }} />
              </Card>
            </Col>
          </Row>

          {/* Error Stats Cards */}
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic title="4xx Errors" value={data.errorStats.error4xx} prefix={<WarningOutlined />} valueStyle={{ color: '#faad14' }} />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic title="Blocked 4xx" value={data.errorStats.blocked4xx} prefix={<StopOutlined />} valueStyle={{ color: '#fa8c16' }} />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic title="5xx Errors" value={data.errorStats.error5xx} prefix={<BugOutlined />} valueStyle={{ color: '#ff4d4f' }} />
              </Card>
            </Col>
          </Row>

          {/* Trend Chart */}
          <Card title="实时 QPS 趋势">
            {data.trend && data.trend.length > 0 ? (
              <TrendChart times={data.trend.map((t) => t.time)} values={data.trend.map((t) => t.value)} />
            ) : (
              <Empty description="No trend data" />
            )}
          </Card>

          {/* Geographic Distribution Map */}
          <Card title="访问地理分布">
            {(data.geo && data.geo.length > 0) || (data.geoProvince && data.geoProvince.length > 0) ? (
              <MapChart chinaData={data.geoProvince || []} worldData={data.geo || []} />
            ) : (
              <Empty description="No geographic data" />
            )}
          </Card>

          {/* Recent Logs Table */}
          <Card title="Recent Logs">
            <Table<DashboardRecentItem>
              columns={columns}
              dataSource={data.recent}
              rowKey="id"
              pagination={{ pageSize: 10 }}
              size="small"
              locale={{ emptyText: <Empty description="No recent logs" /> }}
            />
          </Card>
        </>
      )}
    </Space>
  );
}
