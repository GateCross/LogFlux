import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Button, Form, Input, message, Modal, Select, Space } from 'antd';
import React, { useRef, useState } from 'react';
import { getTemplateList, previewTemplate } from '@/services/notification';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const { TextArea } = Input;

const NotificationTemplate: React.FC = () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [form] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [previewContent, setPreviewContent] = useState('');
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
        id: 'notification.template.delete.confirm',
        defaultMessage: 'Confirm Delete',
      }),
      content: intl.formatMessage(
        {
          id: 'notification.template.delete.content',
          defaultMessage: 'Are you sure you want to delete template "{name}"?',
        },
        { name: record.name },
      ),
      onOk: async () => {
        try {
          // Assume delete API exists
          message.success(
            intl.formatMessage({
              id: 'notification.template.delete.success',
              defaultMessage: 'Template deleted successfully',
            }),
          );
          actionRef.current?.reload();
        } catch (error) {
          message.error(
            intl.formatMessage({
              id: 'notification.template.delete.error',
              defaultMessage: 'Failed to delete template',
            }),
          );
        }
      },
    });
  };

  const handlePreview = async (record: any) => {
    try {
      const res = await previewTemplate({ format: record.format, content: record.content });
      setPreviewContent(res?.data?.content || '');
      setPreviewVisible(true);
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'notification.template.preview.error',
          defaultMessage: 'Failed to preview template',
        }),
      );
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      // Assume create/update APIs exist
      message.success(
        intl.formatMessage({
          id: 'notification.template.save.success',
          defaultMessage: 'Template saved successfully',
        }),
      );
      setModalVisible(false);
      form.resetFields();
      actionRef.current?.reload();
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'notification.template.save.error',
          defaultMessage: 'Failed to save template',
        }),
      );
    } finally {
      setLoading(false);
    }
  };

  const formatOptions = [
    { label: 'Text', value: 'text' },
    { label: 'HTML', value: 'html' },
    { label: 'Markdown', value: 'markdown' },
    { label: 'JSON', value: 'json' },
  ];

  const columns: ProColumns[] = [
    {
      title: intl.formatMessage({ id: 'notification.template.name', defaultMessage: 'Name' }),
      dataIndex: 'name',
      width: 200,
    },
    {
      title: intl.formatMessage({ id: 'notification.template.format', defaultMessage: 'Format' }),
      dataIndex: 'format',
      width: 120,
      valueType: 'select',
      valueEnum: {
        text: { text: 'Text' },
        html: { text: 'HTML' },
        markdown: { text: 'Markdown' },
        json: { text: 'JSON' },
      },
    },
    {
      title: intl.formatMessage({ id: 'notification.template.created.at', defaultMessage: 'Created At' }),
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'notification.template.action', defaultMessage: 'Action' }),
      valueType: 'option',
      width: 200,
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => handlePreview(record)}>
            {intl.formatMessage({ id: 'notification.template.preview', defaultMessage: 'Preview' })}
          </Button>
          <Button type="link" onClick={() => handleEdit(record)}>
            {intl.formatMessage({ id: 'notification.template.edit', defaultMessage: 'Edit' })}
          </Button>
          <Button type="link" danger onClick={() => handleDelete(record)}>
            {intl.formatMessage({ id: 'notification.template.delete', defaultMessage: 'Delete' })}
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
            {intl.formatMessage({ id: 'notification.template.add', defaultMessage: 'Add Template' })}
          </Button>,
        ]}
        request={async (params) => {
          try {
            const res = await getTemplateList();
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
          id: editingRecord ? 'notification.template.edit' : 'notification.template.add',
          defaultMessage: editingRecord ? 'Edit Template' : 'Add Template',
        })}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={loading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label={intl.formatMessage({ id: 'notification.template.name', defaultMessage: 'Name' })}
            name="name"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.template.format', defaultMessage: 'Format' })}
            name="format"
            rules={[{ required: true }]}
          >
            <Select options={formatOptions} />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'notification.template.content', defaultMessage: 'Content' })}
            name="content"
            rules={[{ required: true }]}
          >
            <TextArea rows={10} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={intl.formatMessage({
          id: 'notification.template.preview.title',
          defaultMessage: 'Template Preview',
        })}
        open={previewVisible}
        onCancel={() => setPreviewVisible(false)}
        footer={null}
        width={700}
      >
        <pre style={{ maxHeight: 500, overflow: 'auto', background: '#f5f5f5', padding: 16 }}>
          {previewContent}
        </pre>
      </Modal>
    </PageContainer>
  );
};

export default NotificationTemplate;
