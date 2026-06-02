import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Button, message, Modal, Tag } from 'antd';
import React, { useRef, useState } from 'react';
import { getLogList } from '@/services/notification';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const NotificationLog: React.FC = () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning(
        intl.formatMessage({
          id: 'notification.log.select.warning',
          defaultMessage: 'Please select at least one log',
        }),
      );
      return;
    }
    Modal.confirm({
      title: intl.formatMessage({
        id: 'notification.log.batch.delete.confirm',
        defaultMessage: 'Confirm Batch Delete',
      }),
      content: intl.formatMessage(
        {
          id: 'notification.log.batch.delete.content',
          defaultMessage: 'Are you sure you want to delete {count} logs?',
        },
        { count: selectedRowKeys.length },
      ),
      onOk: async () => {
        try {
          // Assume batch delete API exists
          message.success(
            intl.formatMessage({
              id: 'notification.log.batch.delete.success',
              defaultMessage: 'Logs deleted successfully',
            }),
          );
          setSelectedRowKeys([]);
          actionRef.current?.reload();
        } catch (error) {
          message.error(
            intl.formatMessage({
              id: 'notification.log.batch.delete.error',
              defaultMessage: 'Failed to delete logs',
            }),
          );
        }
      },
    });
  };

  const handleClearAll = () => {
    Modal.confirm({
      title: intl.formatMessage({
        id: 'notification.log.clear.all.confirm',
        defaultMessage: 'Confirm Clear All',
      }),
      content: intl.formatMessage({
        id: 'notification.log.clear.all.content',
        defaultMessage: 'Are you sure you want to clear all notification logs?',
      }),
      okType: 'danger',
      onOk: async () => {
        try {
          // Assume clear all API exists
          message.success(
            intl.formatMessage({
              id: 'notification.log.clear.all.success',
              defaultMessage: 'All logs cleared successfully',
            }),
          );
          setSelectedRowKeys([]);
          actionRef.current?.reload();
        } catch (error) {
          message.error(
            intl.formatMessage({
              id: 'notification.log.clear.all.error',
              defaultMessage: 'Failed to clear logs',
            }),
          );
        }
      },
    });
  };

  const getStatusTag = (status: string) => {
    const map: Record<string, { color: string; text: string }> = {
      success: { color: 'green', text: 'Success' },
      failed: { color: 'red', text: 'Failed' },
      pending: { color: 'default', text: 'Pending' },
    };
    const config = map[status] || { color: 'default', text: status };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  const columns: ProColumns[] = [
    {
      title: intl.formatMessage({ id: 'notification.log.time', defaultMessage: 'Time' }),
      dataIndex: 'timestamp',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'notification.log.channel', defaultMessage: 'Channel' }),
      dataIndex: 'channelName',
      width: 150,
      valueType: 'select',
    },
    {
      title: intl.formatMessage({ id: 'notification.log.rule', defaultMessage: 'Rule' }),
      dataIndex: 'ruleName',
      width: 200,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'notification.log.status', defaultMessage: 'Status' }),
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      valueEnum: {
        success: { text: 'Success', status: 'Success' },
        failed: { text: 'Failed', status: 'Error' },
        pending: { text: 'Pending', status: 'Default' },
      },
      render: (_, record) => getStatusTag(record.status),
    },
    {
      title: intl.formatMessage({ id: 'notification.log.job.status', defaultMessage: 'Job Status' }),
      dataIndex: 'jobStatus',
      width: 120,
      valueType: 'select',
      valueEnum: {
        completed: { text: 'Completed', status: 'Success' },
        running: { text: 'Running', status: 'Processing' },
        failed: { text: 'Failed', status: 'Error' },
      },
    },
    {
      title: intl.formatMessage({ id: 'notification.log.message', defaultMessage: 'Message' }),
      dataIndex: 'message',
      ellipsis: true,
      search: false,
    },
  ];

  return (
    <PageContainer>
      <ProTable
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        toolBarRender={() => [
          <Button key="batch-delete" onClick={handleBatchDelete}>
            {intl.formatMessage({ id: 'notification.log.batch.delete', defaultMessage: 'Batch Delete' })}
          </Button>,
          <Button key="clear-all" danger onClick={handleClearAll}>
            {intl.formatMessage({ id: 'notification.log.clear.all', defaultMessage: 'Clear All' })}
          </Button>,
        ]}
        request={async (params) => {
          try {
            const res = await getLogList({ page: params.current || 1, pageSize: params.pageSize || 20, ...params });
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
    </PageContainer>
  );
};

export default NotificationLog;
