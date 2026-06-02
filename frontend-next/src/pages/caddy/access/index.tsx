import { PageContainer } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import { Button, Card, Form, message, Select, Spin, Switch } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { fetchIPRegionConfig, updateIPRegionConfig, type IPRegionConfig } from '@/services/ip-region';

const CaddyAccess: React.FC = () => {
  const intl = useIntl();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<IPRegionConfig | null>(null);

  const loadConfig = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await fetchIPRegionConfig();
      setConfig(data);
      form.setFieldsValue(data);
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'caddy.access.load.error',
          defaultMessage: 'Failed to load IP region config',
        }),
      );
    } finally {
      setLoading(false);
    }
  }, [form, intl]);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);
      await updateIPRegionConfig(values);
      message.success(
        intl.formatMessage({
          id: 'caddy.access.save.success',
          defaultMessage: 'Config saved successfully',
        }),
      );
      await loadConfig();
    } catch (error) {
      message.error(
        intl.formatMessage({
          id: 'caddy.access.save.error',
          defaultMessage: 'Failed to save config',
        }),
      );
    } finally {
      setSaving(false);
    }
  };

  const countryOptions = [
    { label: 'United States', value: 'US' },
    { label: 'China', value: 'CN' },
    { label: 'Japan', value: 'JP' },
    { label: 'Korea', value: 'KR' },
    { label: 'Germany', value: 'DE' },
    { label: 'United Kingdom', value: 'GB' },
    { label: 'France', value: 'FR' },
    { label: 'Canada', value: 'CA' },
  ];

  return (
    <PageContainer>
      <Card>
        <Spin spinning={loading}>
          <Form form={form} layout="vertical">
            <Form.Item
              label={intl.formatMessage({
                id: 'caddy.access.enable',
                defaultMessage: 'Enable IP Region Restriction',
              })}
              name="enabled"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Form.Item
              label={intl.formatMessage({
                id: 'caddy.access.countries',
                defaultMessage: 'Allowed Countries',
              })}
              name="allowList"
            >
              <Select
                mode="multiple"
                placeholder={intl.formatMessage({
                  id: 'caddy.access.countries.placeholder',
                  defaultMessage: 'Select allowed countries',
                })}
                options={countryOptions}
                style={{ width: '100%' }}
              />
            </Form.Item>

            <Form.Item>
              <Button type="primary" loading={saving} onClick={handleSave}>
                {intl.formatMessage({
                  id: 'caddy.access.save',
                  defaultMessage: 'Save',
                })}
              </Button>
            </Form.Item>
          </Form>
        </Spin>
      </Card>
    </PageContainer>
  );
};

export default CaddyAccess;
