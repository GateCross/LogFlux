/**
 * Security Workbench -- thin wrapper.
 *
 * Delegates to four extracted tab components:
 *   SourceTab / PolicyTab / ObserveTab / OpsTab
 */
import { useState } from 'react';
import { Tabs } from 'antd';
import { PageContainer } from '@ant-design/pro-components';
import { useIntl } from '@umijs/max';
import SourceTab from './components/SourceTab';
import PolicyTab from './components/PolicyTab';
import ObserveTab from './components/ObserveTab';
import OpsTab from './components/OpsTab';

interface SecurityWorkbenchProps {
  defaultTab?: string;
}

export default function SecurityWorkbench({ defaultTab = 'source' }: SecurityWorkbenchProps) {
  const intl = useIntl();
  const [activeTab, setActiveTab] = useState(defaultTab);

  const tabItems = [
    { key: 'source', label: intl.formatMessage({ id: 'route.security_source', defaultMessage: '规则源' }), children: <SourceTab /> },
    { key: 'policy', label: intl.formatMessage({ id: 'route.security_policy', defaultMessage: '策略中心' }), children: <PolicyTab /> },
    { key: 'observe', label: intl.formatMessage({ id: 'route.security_observe', defaultMessage: '监控观测' }), children: <ObserveTab /> },
    { key: 'ops', label: intl.formatMessage({ id: 'route.security_ops', defaultMessage: '发布运维' }), children: <OpsTab /> },
  ];

  return (
    <PageContainer title={intl.formatMessage({ id: 'route.security', defaultMessage: '安全 WAF' })}>
      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} size="large" />
    </PageContainer>
  );
}
