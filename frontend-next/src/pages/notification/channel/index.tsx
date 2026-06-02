import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Button, Form, Input, message, Modal, Select, Space } from 'antd';
import React, { useRef, useState } from 'react';
import {
  getChannelList,
  testChannel,
  createChannel,
  updateChannel,
  deleteChannel,
} from '@/services/notification';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const NotificationChannel: React.FC = () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [form] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const handleAdd = () => {
    setEditingRecord(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setEditingRecord(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = (record: any) => {
    Modal.confirm({
      title: intl.formatMessage({
        id: 'notification.channel.delete.confirm',
        defaultMessage: 'Confirm Delete',
      }),
      content: intl.formatMessage(
        {
          id: 'notification.channel.delete.content',
          defaultMessage: 'Are you sure you want to delete channel "{name}"?',
        },
        { name: record.name },
      ),
      onOk: async () => {
        try {
          const res = await deleteChannel(record.id);
          if (res.error) throw res.error;
          message.success(
            intl.formatMessage({
              id: 'notification.channel.delete.success',
              defaultMessage: 'Channel deleted successfully',
            }),
          );
          actionRef.current?.reload();
        } catch (error) {
          message.error(
            intl.formatMessage({
              id: 'notification.channel.delete.error',
              defaultMessage: 'Failed to delete channel',
            }),
          );
        }
      },
    });
  };

  const handleTest = async (record: any) => {
    try {
      await testChannel({ id: record.id });
      message.success(
        intl.formatMessage({
          id: 'notification.channel.test.success',
          defaultMessage: 'Test message sent successfully',
        }),
      );
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'notification.channel.test.error',
          defaultMessage: 'Failed to send test message',
        }),
      );
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      if (editingRecord) {
        const res = await updateChannel(editingRecord.id, values);
        if (res.error) throw res.error;
      } else {
        const res = await createChannel(values);
        if (res.error) throw res.error;
      }
      message.success(
        intl.formatMessage({
          id: 'notification.channel.save.success',
          defaultMessage: 'Channel saved successfully',
        }),
      );
      setModalVisible(false);
      form.resetFields();
      actionRef.current?.reload();
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'notification.channel.save.error',
          defaultMessage: 'Failed to save channel',
        }),
      );
    } finally {
      setLoading(false);
    }
  };

  const channelTypes = [
    { label: 'Webhook', value: 'webhook' },
    { label: 'Telegram', value: 'telegram' },
    { label: 'Slack', value: 'slack' },
    { label: 'DingTalk', value: 'dingtalk' },
    { label: 'Feishu', value: 'feishu' },
    { label: 'Email', value: 'email' },
  ];

  const columns: ProColumns[] = [
    {
      title: intl.formatMessage({ id: 'notification.channel.name', defaultMessage: 'Name' }),
      dataIndex: 'name',
      width: 200,
    },
    {
      title: intl.formatMessage({ id: 'notification.channel.type', defaultMessage: 'Type' }),
      dataIndex: 'type',
      width: 120,
      valueType: 'select',
      valueEnum: {
        webhook: { text: 'Webhook' },
        telegram: { text: 'Telegram' },
        slack: { text: 'Slack' },
        dingtalk: { text: 'DingTalk' },
        feishu: { text: 'Feishu' },
        email: { text: 'Email' },
      },
    },
    {
      title: intl.formatMessage({ id: 'notification.channel.status', defaultMessage: 'Status' }),
      dataIndex: 'enabled',
      width: 100,
      valueType: 'select',
      valueEnum: {
        true: { text: 'Enabled', status: 'Success' },
        false: { text: 'Disabled', status: 'Default' },
      },
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'notification.channel.created.at', defaultMessage: 'Created At' }),
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'notification.channel.action', defaultMessage: 'Action' }),
      valueType: 'option',
      width: 200,
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => handleTest(record)}>
            {intl.formatMessage({ id: 'notification.channel.test', defaultMessage: 'Test' })}
          </Button>
          <Button type="link" onClick={() => handleEdit(record)}>
            {intl.formatMessage({ id: 'notification.channel.edit', defaultMessage: 'Edit' })}
          </Button>
          <Button type="link" danger onClick={() => handleDelete(record)}>
            {intl.formatMessage({ id: 'notification.channel.delete', defaultMessage: 'Delete' })}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer>
      <ProTable
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        toolBarRender={() => [
          <Button key="add" type="primary" onClick={handleAdd}>
            {intl.formatMessage({ id: 'notification.channel.add', defaultMessage: 'Add Channel' })}
          </Button>,
        ]}
        request={async (params) => {
          try {
            const res = await getChannelList();
            return {
              data: res?.data?.list || [],
              success: true,
              total: res?.data?.list?.length || 0,
            };
          } catch (error) {
            return { data: [], success: false, total: 0 };
          }
        }}
        pagination={{
          defaultPageSize: 10,
          showSizeChanger: true,
        }}
        search={{
          labelWidth: 'auto',
        }}
      />

      <Modal
        title={intl.formatMessage({
          id: editingRecord ? 'notification.channel.edit' : 'notification.channel.add',
          defaultMessage: editingRecord ? 'Edit Channel' : 'Add Channel',
        })}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={loading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label={intl.formatMessage({ id: 'notification.channel.name', defaultMessage: 'Name' })}
            name="name"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.channel.type', defaultMessage: 'Type' })}
            name="type"
            rules={[{ required: true }]}
          >
            <Select options={channelTypes} />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.channel.config', defaultMessage: 'Config' })}
            name="config"
            rules={[{ required: true }]}
          >
            <Input.TextArea rows={6} placeholder="JSON config" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
};

export default NotificationChannel;
