import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Modal, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import { fetchCaddyLogs } from '@/services/caddy';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const getStatusColor = (status: number) => {
  if (status >= 200 && status < 300) return 'green';
  if (status >= 300 && status < 400) return 'blue';
  if (status >= 400 && status < 500) return 'orange';
  if (status >= 500) return 'red';
  return 'default';
};

const CaddyLog: React.FC = () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [selectedRecord, setSelectedRecord] = useState<any>(null);
  const [modalVisible, setModalVisible] = useState(false);

  const handleRowClick = (record: any) => {
    setSelectedRecord(record);
    setModalVisible(true);
  };

  const columns: ProColumns[] = [
    {
      title: intl.formatMessage({ id: 'caddy.log.time', defaultMessage: 'Time' }),
      dataIndex: 'timestamp',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'caddy.log.host', defaultMessage: 'Host' }),
      dataIndex: 'host',
      width: 200,
      ellipsis: true,
    },
    {
      title: intl.formatMessage({ id: 'caddy.log.method', defaultMessage: 'Method' }),
      dataIndex: 'method',
      width: 80,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'caddy.log.path', defaultMessage: 'Path' }),
      dataIndex: 'path',
      ellipsis: true,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'caddy.log.status', defaultMessage: 'Status' }),
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      valueEnum: {
        200: { text: '200', status: 'Success' },
        301: { text: '301', status: 'Processing' },
        302: { text: '302', status: 'Processing' },
        400: { text: '400', status: 'Warning' },
        401: { text: '401', status: 'Warning' },
        403: { text: '403', status: 'Warning' },
        404: { text: '404', status: 'Warning' },
        500: { text: '500', status: 'Error' },
        502: { text: '502', status: 'Error' },
        503: { text: '503', status: 'Error' },
      },
      render: (_, record) => (
        <Tag color={getStatusColor(record.status)}>{record.status}</Tag>
      ),
    },
    {
      title: intl.formatMessage({ id: 'caddy.log.duration', defaultMessage: 'Duration' }),
      dataIndex: 'duration',
      width: 100,
      search: false,
      render: (_, record) => `${(record.duration / 1000).toFixed(2)}s`,
    },
    {
      title: intl.formatMessage({ id: 'caddy.log.keyword', defaultMessage: 'Keyword' }),
      dataIndex: 'keyword',
      hideInTable: true,
    },
    {
      title: intl.formatMessage({ id: 'caddy.log.time.range', defaultMessage: 'Time Range' }),
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
      <ProTable
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        onRow={(record) => ({
          onClick: () => handleRowClick(record),
        })}
        request={async (params) => {
          try {
            const res = await fetchCaddyLogs({ page: params.current || 1, pageSize: params.pageSize || 20, ...params });
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
      />

      <Modal
        title={intl.formatMessage({
          id: 'caddy.log.detail',
          defaultMessage: 'Log Detail',
        })}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={700}
      >
        {selectedRecord && (
          <pre style={{ maxHeight: 500, overflow: 'auto', background: '#f5f5f5', padding: 16 }}>
            {JSON.stringify(selectedRecord, null, 2)}
          </pre>
        )}
      </Modal>
    </PageContainer>
  );
};

export default CaddyLog;
