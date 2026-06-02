/**
 * Dashboard Page (Task 13.1)
 *
 * Statistics overview, error stats, trend chart, geographic distribution map, and recent logs.
 * Fully internationalized — all user-facing strings go through react-intl.
 */
import { useEffect, useState, useCallback, useMemo } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Spin, Empty, Alert, Select, Space, Typography } from 'antd';
import { useIntl } from '@umijs/max';
import {
  ArrowUpOutlined,
  EyeOutlined,
  UserOutlined,
  StopOutlined,
  ThunderboltOutlined,
  WarningOutlined,
  BugOutlined,
  GlobalOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { fetchDashboardSummary } from '@/services/dashboard';
import type { DashboardSummaryResp, DashboardRecentItem } from '@/services/dashboard';
import TrendChart from '@/components/charts/TrendChart';
import MapChart from '@/components/charts/MapChart';

const TIME_RANGE_VALUES = [3600, 21600, 86400, 604800, 2592000] as const;

export default function DashboardPage() {
  const intl = useIntl();

  /** Shorthand translation helper — memoised so it only changes when `intl` changes. */
  const t = useCallback(
    (id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }),
    [intl],
  );

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<DashboardSummaryResp | null>(null);
  const [rangeSeconds, setRangeSeconds] = useState<number>(86400);

  // --- Time-range preset options (labels need i18n, so build inside component) ---
  const timeRangeOptions = useMemo(
    () => [
      { label: t('dashboard.range.1h', '1 Hour'), value: TIME_RANGE_VALUES[0] },
      { label: t('dashboard.range.6h', '6 Hours'), value: TIME_RANGE_VALUES[1] },
      { label: t('dashboard.range.24h', '24 Hours'), value: TIME_RANGE_VALUES[2] },
      { label: t('dashboard.range.7d', '7 Days'), value: TIME_RANGE_VALUES[3] },
      { label: t('dashboard.range.30d', '30 Days'), value: TIME_RANGE_VALUES[4] },
    ],
    [t],
  );

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

  // --- Recent logs table columns (memoised to avoid re-creation on every render) ---
  const columns: ColumnsType<DashboardRecentItem> = useMemo(
    () => [
      {
        title: t('dashboard.recent.time', 'Time'),
        dataIndex: 'logTime',
        key: 'logTime',
        width: 180,
        render: (text: string) => (text ? new Date(text).toLocaleString() : '-'),
      },
      {
        title: t('dashboard.recent.method', 'Method'),
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
      {
        title: t('dashboard.recent.uri', 'URI'),
        dataIndex: 'uri',
        key: 'uri',
        ellipsis: true,
      },
      {
        title: t('dashboard.recent.status', 'Status'),
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
      {
        title: t('dashboard.recent.remoteIp', 'Remote IP'),
        dataIndex: 'remoteIp',
        key: 'remoteIp',
        width: 150,
      },
      {
        title: t('dashboard.recent.country', 'Country'),
        dataIndex: 'country',
        key: 'country',
        width: 120,
      },
    ],
    [t],
  );

  // --- Render ---
  return (
    <Space direction="vertical" size="middle" style={{ display: 'flex' }}>
      {/* Header with title and time range selector */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          {t('dashboard.title', 'Dashboard')}
        </Typography.Title>
        <Select
          value={rangeSeconds}
          onChange={(v) => setRangeSeconds(v)}
          style={{ width: 140 }}
          options={timeRangeOptions}
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
          message={t('common.error', 'Error')}
          description={error}
          type="error"
          showIcon
          closable
          action={<a onClick={() => loadData(rangeSeconds)}>{t('common.retry', 'Retry')}</a>}
        />
      )}

      {/* Empty state */}
      {!loading && !error && !data && (
        <Card>
          <Empty description={t('dashboard.empty', 'No dashboard data available')} />
        </Card>
      )}

      {/* Data loaded */}
      {!loading && !error && data && (
        <>
          {/* Summary Stats Cards */}
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic
                  title={t('dashboard.stats.requests', 'Requests')}
                  value={data.stats.requests}
                  prefix={<ArrowUpOutlined />}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic
                  title={t('dashboard.stats.pv', 'PV')}
                  value={data.stats.pv}
                  prefix={<EyeOutlined />}
                  valueStyle={{ color: '#722ed1' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic
                  title={t('dashboard.stats.uv', 'UV')}
                  value={data.stats.uv}
                  prefix={<UserOutlined />}
                  valueStyle={{ color: '#13c2c2' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic
                  title={t('dashboard.stats.uniqueIp', 'Unique IP')}
                  value={data.stats.uniqueIp}
                  prefix={<GlobalOutlined />}
                  valueStyle={{ color: '#52c41a' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic
                  title={t('dashboard.stats.blocked', 'Blocked')}
                  value={data.stats.blocked}
                  prefix={<StopOutlined />}
                  valueStyle={{ color: '#faad14' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={4}>
              <Card hoverable>
                <Statistic
                  title={t('dashboard.stats.attackIp', 'Attack IP')}
                  value={data.stats.attackIp}
                  prefix={<ThunderboltOutlined />}
                  valueStyle={{ color: '#ff4d4f' }}
                />
              </Card>
            </Col>
          </Row>

          {/* Error Stats Cards */}
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title={t('dashboard.errorStats.4xx', '4xx Errors')}
                  value={data.errorStats.error4xx}
                  prefix={<WarningOutlined />}
                  valueStyle={{ color: '#faad14' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title={t('dashboard.errorStats.blocked4xx', 'Blocked 4xx')}
                  value={data.errorStats.blocked4xx}
                  prefix={<StopOutlined />}
                  valueStyle={{ color: '#fa8c16' }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={8}>
              <Card>
                <Statistic
                  title={t('dashboard.errorStats.5xx', '5xx Errors')}
                  value={data.errorStats.error5xx}
                  prefix={<BugOutlined />}
                  valueStyle={{ color: '#ff4d4f' }}
                />
              </Card>
            </Col>
          </Row>

          {/* Trend Chart */}
          <Card title={t('dashboard.trend.qps', '实时 QPS 趋势')}>
            {data.trend && data.trend.length > 0 ? (
              <TrendChart times={data.trend.map((item) => item.time)} values={data.trend.map((item) => item.value)} />
            ) : (
              <Empty description={t('dashboard.trend.empty', 'No trend data')} />
            )}
          </Card>

          {/* Geographic Distribution Map */}
          <Card title={t('dashboard.geo.title', '访问地理分布')}>
            {(data.geo && data.geo.length > 0) || (data.geoProvince && data.geoProvince.length > 0) ? (
              <MapChart chinaData={data.geoProvince || []} worldData={data.geo || []} />
            ) : (
              <Empty description={t('dashboard.geo.empty', 'No geographic data')} />
            )}
          </Card>

          {/* Recent Logs Table */}
          <Card title={t('dashboard.recent.title', 'Recent Logs')}>
            <Table<DashboardRecentItem>
              columns={columns}
              dataSource={data.recent}
              rowKey="id"
              pagination={{ pageSize: 10 }}
              size="small"
              locale={{ emptyText: <Empty description={t('dashboard.recent.empty', 'No recent logs')} /> }}
            />
          </Card>
        </>
      )}
    </Space>
  );
}
