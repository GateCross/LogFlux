import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Modal, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import { fetchCronLogList } from '@/services/cron';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const getStatusTag = (status: string) => {
  const map: Record<string, { color: string; text: string }> = {
    success: { color: 'green', text: 'Success' },
    failed: { color: 'red', text: 'Failed' },
    running: { color: 'blue', text: 'Running' },
    pending: { color: 'default', text: 'Pending' },
  };
  const config = map[status] || { color: 'default', text: status };
  return <Tag color={config.color}>{config.text}</Tag>;
};

const CronLog: React.FC = () => {
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
      title: intl.formatMessage({ id: 'cron.log.task.id', defaultMessage: 'Task ID' }),
      dataIndex: 'taskId',
      width: 120,
      valueType: 'select',
    },
    {
      title: intl.formatMessage({ id: 'cron.log.task.name', defaultMessage: 'Task Name' }),
      dataIndex: 'taskName',
      width: 200,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'cron.log.start.time', defaultMessage: 'Start Time' }),
      dataIndex: 'startTime',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'cron.log.end.time', defaultMessage: 'End Time' }),
      dataIndex: 'endTime',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'cron.log.duration', defaultMessage: 'Duration' }),
      dataIndex: 'duration',
      width: 100,
      search: false,
      render: (_, record) => {
        if (!record.duration) return '-';
        return `${(record.duration / 1000).toFixed(2)}s`;
      },
    },
    {
      title: intl.formatMessage({ id: 'cron.log.status', defaultMessage: 'Status' }),
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      valueEnum: {
        success: { text: 'Success', status: 'Success' },
        failed: { text: 'Failed', status: 'Error' },
        running: { text: 'Running', status: 'Processing' },
        pending: { text: 'Pending', status: 'Default' },
      },
      render: (_, record) => getStatusTag(record.status),
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
            const res = await fetchCronLogList(params);
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
        }}
      />

      <Modal
        title={intl.formatMessage({
          id: 'cron.log.detail',
          defaultMessage: 'Execution Log Detail',
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

export default CronLog;
