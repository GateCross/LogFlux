import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import {
  Button,
  Form,
  Input,
  message,
  Modal,
  Radio,
  Space,
  Switch,
  Upload,
} from 'antd';
import React, { useRef, useState } from 'react';
import {
  createCronTask,
  deleteCronTask,
  fetchCronTaskList,
  triggerCronTask,
  updateCronTask,
} from '@/services/cron';
import { UploadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const { TextArea } = Input;

const CronTasks: React.FC = () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [form] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [scriptMode, setScriptMode] = useState<'inline' | 'upload'>('inline');

  const handleAdd = () => {
    setEditingRecord(null);
    form.resetFields();
    setScriptMode('inline');
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setEditingRecord(record);
    form.setFieldsValue(record);
    setScriptMode(record.script ? 'inline' : 'upload');
    setModalVisible(true);
  };

  const handleDelete = (record: any) => {
    Modal.confirm({
      title: intl.formatMessage({
        id: 'cron.task.delete.confirm',
        defaultMessage: 'Confirm Delete',
      }),
      content: intl.formatMessage(
        {
          id: 'cron.task.delete.content',
          defaultMessage: 'Are you sure you want to delete task "{name}"?',
        },
        { name: record.name },
      ),
      onOk: async () => {
        try {
          await deleteCronTask(record.id);
          message.success(
            intl.formatMessage({
              id: 'cron.task.delete.success',
              defaultMessage: 'Task deleted successfully',
            }),
          );
          actionRef.current?.reload();
        } catch (error) {
          message.error(
            intl.formatMessage({
              id: 'cron.task.delete.error',
              defaultMessage: 'Failed to delete task',
            }),
          );
        }
      },
    });
  };

  const handleTrigger = async (record: any) => {
    try {
      await triggerCronTask(record.id);
      message.success(
        intl.formatMessage({
          id: 'cron.task.trigger.success',
          defaultMessage: 'Task triggered successfully',
        }),
      );
      actionRef.current?.reload();
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'cron.task.trigger.error',
          defaultMessage: 'Failed to trigger task',
        }),
      );
    }
  };

  const handleToggleEnable = async (record: any, checked: boolean) => {
    try {
      await updateCronTask(record.id, { status: checked ? 1 : 0 });
      message.success(
        intl.formatMessage({
          id: 'cron.task.toggle.success',
          defaultMessage: 'Task status updated',
        }),
      );
      actionRef.current?.reload();
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'cron.task.toggle.error',
          defaultMessage: 'Failed to update task status',
        }),
      );
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      if (editingRecord) {
        await updateCronTask(editingRecord.id, values);
        message.success(
          intl.formatMessage({
            id: 'cron.task.update.success',
            defaultMessage: 'Task updated successfully',
          }),
        );
      } else {
        await createCronTask(values);
        message.success(
          intl.formatMessage({
            id: 'cron.task.create.success',
            defaultMessage: 'Task created successfully',
          }),
        );
      }
      setModalVisible(false);
      form.resetFields();
      actionRef.current?.reload();
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'cron.task.save.error',
          defaultMessage: 'Failed to save task',
        }),
      );
    } finally {
      setLoading(false);
    }
  };

  const columns: ProColumns[] = [
    {
      title: intl.formatMessage({ id: 'cron.task.name', defaultMessage: 'Name' }),
      dataIndex: 'name',
      width: 200,
    },
    {
      title: intl.formatMessage({ id: 'cron.task.cron', defaultMessage: 'Cron Expression' }),
      dataIndex: 'cronExpression',
      width: 150,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'cron.task.status', defaultMessage: 'Status' }),
      dataIndex: 'enabled',
      width: 100,
      valueType: 'select',
      valueEnum: {
        true: { text: 'Enabled', status: 'Success' },
        false: { text: 'Disabled', status: 'Default' },
      },
      render: (_, record) => (
        <Switch
          checked={record.enabled}
          onChange={(checked) => handleToggleEnable(record, checked)}
        />
      ),
    },
    {
      title: intl.formatMessage({ id: 'cron.task.last.run', defaultMessage: 'Last Run' }),
      dataIndex: 'lastRunAt',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'cron.task.next.run', defaultMessage: 'Next Run' }),
      dataIndex: 'nextRunAt',
      valueType: 'dateTime',
      width: 180,
      search: false,
    },
    {
      title: intl.formatMessage({ id: 'cron.task.action', defaultMessage: 'Action' }),
      valueType: 'option',
      width: 200,
      render: (_, record) => (
        <Space>
          <Button type="link" onClick={() => handleTrigger(record)}>
            {intl.formatMessage({ id: 'cron.task.trigger', defaultMessage: 'Trigger' })}
          </Button>
          <Button type="link" onClick={() => handleEdit(record)}>
            {intl.formatMessage({ id: 'cron.task.edit', defaultMessage: 'Edit' })}
          </Button>
          <Button type="link" danger onClick={() => handleDelete(record)}>
            {intl.formatMessage({ id: 'cron.task.delete', defaultMessage: 'Delete' })}
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
            {intl.formatMessage({ id: 'cron.task.add', defaultMessage: 'Add Task' })}
          </Button>,
        ]}
        request={async (params) => {
          try {
            const res = await fetchCronTaskList(params);
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
          defaultPageSize: 10,
          showSizeChanger: true,
        }}
        search={{
          labelWidth: 'auto',
        }}
      />

      <Modal
        title={intl.formatMessage({
          id: editingRecord ? 'cron.task.edit' : 'cron.task.add',
          defaultMessage: editingRecord ? 'Edit Task' : 'Add Task',
        })}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        confirmLoading={loading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label={intl.formatMessage({ id: 'cron.task.name', defaultMessage: 'Name' })}
            name="name"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'cron.task.cron', defaultMessage: 'Cron Expression' })}
            name="cronExpression"
            rules={[{ required: true }]}
          >
            <Input placeholder="* * * * *" />
          </Form.Item>
          <Form.Item
            label={intl.formatMessage({ id: 'cron.task.script.mode', defaultMessage: 'Script Mode' })}
          >
            <Radio.Group value={scriptMode} onChange={(e) => setScriptMode(e.target.value)}>
              <Radio value="inline">Inline Text</Radio>
              <Radio value="upload">File Upload</Radio>
            </Radio.Group>
          </Form.Item>
          {scriptMode === 'inline' ? (
            <Form.Item
              label={intl.formatMessage({ id: 'cron.task.script', defaultMessage: 'Script' })}
              name="script"
              rules={[{ required: true }]}
            >
              <TextArea rows={8} />
            </Form.Item>
          ) : (
            <Form.Item
              label={intl.formatMessage({ id: 'cron.task.script.file', defaultMessage: 'Script File' })}
              name="scriptFile"
              rules={[{ required: true }]}
            >
              <Upload>
                <Button icon={<UploadOutlined />}>
                  {intl.formatMessage({ id: 'cron.task.upload', defaultMessage: 'Upload' })}
                </Button>
              </Upload>
            </Form.Item>
          )}
        </Form>
      </Modal>
    </PageContainer>
  );
};

export default CronTasks;
