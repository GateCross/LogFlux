import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Switch, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import { fetchSystemLogs } from '@/services/system-log';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const getLevelColor = (level: string) => {
  const map: Record<string, string> = {
    info: 'blue',
    warn: 'orange',
    error: 'red',
    debug: 'default',
  };
  return map[level?.toLowerCase()] || 'default';
};

const CaddySystemLog: React.FC = () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [autoRefresh, setAutoRefresh] = useState(false);

  const columns: ProColumns[] = [
    {
      title: intl.formatMessage({ id: 'system.log.time', defaultMessage: 'Time' }),
      dataIndex: 'timestamp',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'system.log.source', defaultMessage: 'Source' }),
      dataIndex: 'source',
      width: 120,
      valueType: 'select',
      valueEnum: {
        caddy: { text: 'Caddy' },
        system: { text: 'System' },
        app: { text: 'Application' },
      },
    },
    {
      title: intl.formatMessage({ id: 'system.log.level', defaultMessage: 'Level' }),
      dataIndex: 'level',
      width: 100,
      valueType: 'select',
      valueEnum: {
        info: { text: 'Info', status: 'Processing' },
        warn: { text: 'Warn', status: 'Warning' },
        error: { text: 'Error', status: 'Error' },
        debug: { text: 'Debug', status: 'Default' },
      },
      render: (_, record) => (
        <Tag color={getLevelColor(record.level)}>{record.level}</Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'system.log.message', defaultMessage: 'Message' }),
      dataIndex: 'message',
      ellipsis: true,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'system.log.keyword', defaultMessage: 'Keyword' }),
      dataIndex: 'keyword',
      hideInTable: true,
    },
    {
      title: intl.formatMessage({ id: 'system.log.time.range', defaultMessage: 'Time Range' }),
      dataIndex: 'timeRange',
      valueType: 'dateTimeRange',
      hideInTable: true,
      search: {
        transform: (value: any) => ({
          startTime: value[0],
          endTime: value[1],
        }),
      },
    },
  ];

  return (
    <PageContainer>
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 8 }}>
        <span>{intl.formatMessage({ id: 'system.log.auto.refresh', defaultMessage: 'Auto Refresh' })}:</span>
        <Switch checked={autoRefresh} onChange={setAutoRefresh} />
      </div>
      <ProTable
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        request={async (params) => {
          try {
            const res = await fetchSystemLogs({ page: params.current || 1, pageSize: params.pageSize || 20, ...params });
            return {
              data: res?.data?.list || [],
              success: true,
              total: res?.data?.total || 0,
            };
          } catch (error) {
            return { data: [], success: false, total: 0 };
          }
        }}
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: true,
          showQuickJumper: true,
        }}
        search={{
          labelWidth: 'auto',
          defaultCollapsed: false,
        }}
        expandable={{
          expandedRowRender: (record) => (
            <pre style={{ margin: 0, background: '#f5f5f5', padding: 16 }}>
              {JSON.stringify(record, null, 2)}
            </pre>
          ),
        }}
        polling={autoRefresh ? 5000 : undefined}
      />
    </PageContainer>
  );
};

export default CaddySystemLog;
