import { PageContainer, ProTable } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import {
  Button,
  Card,
  Input,
  message,
  Modal,
  Select,
  Space,
  Spin,
} from 'antd';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  fetchCaddyConfigHistory,
  fetchCaddyServers,
  previewCaddyConfig,
  rollbackCaddyConfig,
  updateCaddyConfigRaw,
} from '@/services/caddy';
import type { ActionType, ProColumns } from '@ant-design/pro-components';

const { TextArea } = Input;

const CaddyConfig: React.FC = () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [servers, setServers] = useState<any[]>([]);
  const [selectedServer, setSelectedServer] = useState<string>('');
  const [configContent, setConfigContent] = useState<string>('');
  const [previewContent, setPreviewContent] = useState<string>('');
  const [previewVisible, setPreviewVisible] = useState(false);

  const loadServers = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await fetchCaddyServers();
      setServers(data?.list || []);
      if (data?.list && data.list.length > 0) {
        setSelectedServer(String(data.list[0].id || data.list[0].name));
      }
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'caddy.config.load.servers.error',
          defaultMessage: 'Failed to load servers',
        }),
      );
    } finally {
      setLoading(false);
    }
  }, [intl]);

  useEffect(() => {
    loadServers();
  }, [loadServers]);

  const handlePreview = async () => {
    if (!selectedServer) {
      message.warning(
        intl.formatMessage({
          id: 'caddy.config.select.server',
          defaultMessage: 'Please select a server',
        }),
      );
      return;
    }
    setPreviewing(true);
    try {
      const { data } = await previewCaddyConfig(Number(selectedServer), {
        config: configContent,
      });
      setPreviewContent(data?.config || '');
      setPreviewVisible(true);
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'caddy.config.preview.error',
          defaultMessage: 'Failed to preview config',
        }),
      );
    } finally {
      setPreviewing(false);
    }
  };

  const handleSave = async () => {
    if (!selectedServer) {
      message.warning(
        intl.formatMessage({
          id: 'caddy.config.select.server',
          defaultMessage: 'Please select a server',
        }),
      );
      return;
    }
    setSaving(true);
    try {
      await updateCaddyConfigRaw(Number(selectedServer), configContent);
      message.success(
        intl.formatMessage({
          id: 'caddy.config.save.success',
          defaultMessage: 'Config saved successfully',
        }),
      );
      actionRef.current?.reload();
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'caddy.config.save.error',
          defaultMessage: 'Failed to save config',
        }),
      );
    } finally {
      setSaving(false);
    }
  };

  const handleRollback = async (record: any) => {
    Modal.confirm({
      title: intl.formatMessage({
        id: 'caddy.config.rollback.confirm',
        defaultMessage: 'Confirm Rollback',
      }),
      content: intl.formatMessage({
        id: 'caddy.config.rollback.content',
        defaultMessage: 'Are you sure you want to rollback to this version?',
      }),
      onOk: async () => {
        try {
          await rollbackCaddyConfig(Number(selectedServer), record.id);
          message.success(
            intl.formatMessage({
              id: 'caddy.config.rollback.success',
              defaultMessage: 'Rollback successful',
            }),
          );
          actionRef.current?.reload();
        } catch (error) {
          message.error(
            intl.formatMessage({
              id: 'caddy.config.rollback.error',
              defaultMessage: 'Rollback failed',
            }),
          );
        }
      },
    });
  };

  const columns: ProColumns[] = [
    {
      title: intl.formatMessage({ id: 'caddy.config.version', defaultMessage: 'Version' }),
      dataIndex: 'version',
      width: 100,
    },
    {
      title: intl.formatMessage({ id: 'caddy.config.updated.at', defaultMessage: 'Updated At' }),
      dataIndex: 'updatedAt',
      valueType: 'dateTime',
      width: 180,
    },
    {
      title: intl.formatMessage({ id: 'caddy.config.updated.by', defaultMessage: 'Updated By' }),
      dataIndex: 'updatedBy',
      width: 120,
    },
    {
      title: intl.formatMessage({ id: 'caddy.config.action', defaultMessage: 'Action' }),
      valueType: 'option',
      width: 100,
      render: (_, record) => (
        <Button type="link" onClick={() => handleRollback(record)}>
          {intl.formatMessage({ id: 'caddy.config.rollback', defaultMessage: 'Rollback' })}
        </Button>
      ),
    },
  ];

  return (
    <PageContainer>
      <Spin spinning={loading}>
        <Space direction="vertical" style={{ width: '100%' }} size="large">
          <Card
            title={intl.formatMessage({
              id: 'caddy.config.editor',
              defaultMessage: 'Config Editor',
            })}
          >
            <Space direction="vertical" style={{ width: '100%' }} size="middle">
              <Select
                placeholder={intl.formatMessage({
                  id: 'caddy.config.select.server',
                  defaultMessage: 'Select Server',
                })}
                value={selectedServer}
                onChange={setSelectedServer}
                style={{ width: 300 }}
                options={servers.map((s) => ({
                  label: s.name || s.id,
                  value: s.id || s.name,
                }))}
              />
              <TextArea
                rows={15}
                value={configContent}
                onChange={(e) => setConfigContent(e.target.value)}
                placeholder={intl.formatMessage({
                  id: 'caddy.config.placeholder',
                  defaultMessage: 'Enter raw config here...',
                })}
              />
              <Space>
                <Button
                  type="primary"
                  loading={saving}
                  onClick={handleSave}
                >
                  {intl.formatMessage({ id: 'caddy.config.save', defaultMessage: 'Save' })}
                </Button>
                <Button loading={previewing} onClick={handlePreview}>
                  {intl.formatMessage({ id: 'caddy.config.preview', defaultMessage: 'Preview' })}
                </Button>
              </Space>
            </Space>
          </Card>

          <Card
            title={intl.formatMessage({
              id: 'caddy.config.history',
              defaultMessage: 'Config History',
            })}
          >
            <ProTable
              actionRef={actionRef}
              rowKey="version"
              columns={columns}
              search={false}
              request={async (params) => {
                if (!selectedServer) {
                  return { data: [], success: true, total: 0 };
                }
                try {
                  const res = await fetchCaddyConfigHistory(Number(selectedServer), {
                    page: params.current || 1,
                    pageSize: params.pageSize || 10,
                  });
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
            />
          </Card>
        </Space>
      </Spin>

      <Modal
        title={intl.formatMessage({
          id: 'caddy.config.preview.title',
          defaultMessage: 'Config Preview',
        })}
        open={previewVisible}
        onCancel={() => setPreviewVisible(false)}
        footer={null}
        width={800}
      >
        <pre style={{ maxHeight: 500, overflow: 'auto', background: '#f5f5f5', padding: 16 }}>
          {previewContent}
        </pre>
      </Modal>
    </PageContainer>
  );
};

export default CaddyConfig;
