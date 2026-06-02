import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Button, Form, Input, InputNumber, message, Modal, Select, Space } from 'antd';
import React, { useRef, useState } from 'react';
import { getRuleList, createRule, updateRule, deleteRule } from '@/services/notification';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const { TextArea } = Input;

const NotificationRule: React.FC = () => {
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
        id: 'notification.rule.delete.confirm',
        defaultMessage: 'Confirm Delete',
      }),
      content: intl.formatMessage(
        {
          id: 'notification.rule.delete.content',
          defaultMessage: 'Are you sure you want to delete rule "{name}"?',
        },
        { name: record.name },
      ),
      onOk: async () => {
        try {
          const res = await deleteRule(record.id);
          if (res.error) throw res.error;
          message.success(
            intl.formatMessage({
              id: 'notification.rule.delete.success',
              defaultMessage: 'Rule deleted successfully',
            }),
          );
          actionRef.current?.reload();
        } catch (error) {
          message.error(
            intl.formatMessage({
              id: 'notification.rule.delete.error',
              defaultMessage: 'Failed to delete rule',
            }),
          );
        }
      },
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      if (editingRecord) {
        const res = await updateRule(editingRecord.id, values);
        if (res.error) throw res.error;
      } else {
        const res = await createRule(values);
        if (res.error) throw res.error;
      }
      message.success(
        intl.formatMessage({
          id: 'notification.rule.save.success',
          defaultMessage: 'Rule saved successfully',
        }),
      );
      setModalVisible(false);
      form.resetFields();
      actionRef.current?.reload();
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'notification.rule.save.error',
          defaultMessage: 'Failed to save rule',
        }),
      );
    } finally {
      setLoading(false);
    }
  };

  const ruleTypes = [
    { label: 'Threshold', value: 'threshold' },
    { label: 'Pattern', value: 'pattern' },
    { label: 'Anomaly', value: 'anomaly' },
  ];

  const eventTypes = [
    { label: 'Log Event', value: 'log_event' },
    { label: 'Metric Event', value: 'metric_event' },
    { label: 'System Event', value: 'system_event' },
  ];

  const columns: ProColumns[] = [
    {
      title: intl.formatMessage({ id: 'notification.rule.name', defaultMessage: 'Name' }),
      dataIndex: 'name',
      width: 200,
    },
    {
      title: intl.formatMessage({ id: 'notification.rule.type', defaultMessage: 'Rule Type' }),
      dataIndex: 'ruleType',
      width: 120,
      valueType: 'select',
      valueEnum: {
        threshold: { text: 'Threshold' },
        pattern: { text: 'Pattern' },
        anomaly: { text: 'Anomaly' },
      },
    },
    {
      title: intl.formatMessage({ id: 'notification.rule.event.type', defaultMessage: 'Event Type' }),
      dataIndex: 'eventType',
      width: 120,
      valueType: 'select',
      valueEnum: {
        log_event: { text: 'Log Event' },
        metric_event: { text: 'Metric Event' },
        system_event: { text: 'System Event' },
      },
    },
    {
      title: intl.formatMessage({ id: 'notification.rule.channels', defaultMessage: 'Channels' }),
      dataIndex: 'channelIds',
      width: 200,
      search: false,
      render: (_, record) => (record.channelIds || []).join(', '),
    },
    {
      title: intl.formatMessage({ id: 'notification.rule.silence', defaultMessage: 'Silence (min)' }),
      dataIndex: 'silenceDuration',
      width: 120,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'notification.rule.action', defaultMessage: 'Action' }),
      valueType: 'option',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => handleEdit(record)}>
            {intl.formatMessage({ id: 'notification.rule.edit', defaultMessage: 'Edit' })}
          </Button>
          <Button type="link" danger onClick={() => handleDelete(record)}>
            {intl.formatMessage({ id: 'notification.rule.delete', defaultMessage: 'Delete' })}
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
            {intl.formatMessage({ id: 'notification.rule.add', defaultMessage: 'Add Rule' })}
          </Button>,
        ]}
        request={async (params) => {
          try {
            const res = await getRuleList();
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
          id: editingRecord ? 'notification.rule.edit' : 'notification.rule.add',
          defaultMessage: editingRecord ? 'Edit Rule' : 'Add Rule',
        })}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={loading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label={intl.formatMessage({ id: 'notification.rule.name', defaultMessage: 'Name' })}
            name="name"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.rule.type', defaultMessage: 'Rule Type' })}
            name="ruleType"
            rules={[{ required: true }]}
          >
            <Select options={ruleTypes} />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.rule.event.type', defaultMessage: 'Event Type' })}
            name="eventType"
            rules={[{ required: true }]}
          >
            <Select options={eventTypes} />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.rule.condition', defaultMessage: 'Condition' })}
            name="condition"
            rules={[{ required: true }]}
          >
            <TextArea rows={4} placeholder="JSON condition" />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.rule.channels', defaultMessage: 'Channel IDs' })}
            name="channelIds"
            rules={[{ required: true }]}
          >
            <Select mode="tags" placeholder="Enter channel IDs" />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.rule.template', defaultMessage: 'Template' })}
            name="template"
          >
            <Input />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.rule.silence', defaultMessage: 'Silence Duration (min)' })}
            name="silenceDuration"
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
};

export default NotificationRule;
