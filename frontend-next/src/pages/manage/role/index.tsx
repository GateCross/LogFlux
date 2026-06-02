/**
 * Role Management Page (Task 14.1)
 *
 * Displays roles in a table with permission editing via Modal.
 * Uses fetchGetRoleList() and fetchUpdateRolePermissions() from role service.
 */
import { useEffect, useState, useCallback } from 'react';
import { Table, Button, Modal, Space, message, Card, Tag, Form, Input, Checkbox } from 'antd';
import { PageContainer } from '@ant-design/pro-components';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import type { ColumnsType } from 'antd/es/table';
import { fetchGetRoleList, fetchUpdateRolePermissions } from '@/services/role';

/** Available permission keys for the system. */
const ALL_PERMISSIONS = [
  'dashboard:view',
  'user:list',
  'user:create',
  'user:update',
  'user:delete',
  'role:list',
  'role:update',
  'role:delete',
  'menu:list',
  'menu:create',
  'menu:update',
  'menu:delete',
  'security:view',
  'security:manage',
  'system:log',
  'notification:manage',
];

export default function RoleManagementPage() {
  const intl = useIntl();

  const [loading, setLoading] = useState(false);
  const [roles, setRoles] = useState<Api.Role.RoleItem[]>([]);
  const [editingRole, setEditingRole] = useState<Api.Role.RoleItem | null>(null);
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [permModalVisible, setPermModalVisible] = useState(false);

  const loadRoles = useCallback(async () => {
    setLoading(true);
    const { data, error } = await fetchGetRoleList();
    if (error) {
      message.error(error.message || 'Failed to load roles');
    } else if (data) {
      setRoles(data.list || []);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    loadRoles();
  }, [loadRoles]);

  const handleEditPermissions = (role: Api.Role.RoleItem) => {
    setEditingRole(role);
    setSelectedPermissions([...(role.permissions || [])]);
    setPermModalVisible(true);
  };

  const handlePermissionChange = (checkedValues: string[]) => {
    setSelectedPermissions(checkedValues);
  };

  const handleSavePermissions = async () => {
    if (!editingRole) return;
    setSaving(true);
    const { error } = await fetchUpdateRolePermissions(editingRole.id, selectedPermissions);
    if (error) {
      message.error(error.message || 'Failed to update permissions');
    } else {
      message.success('Permissions updated successfully');
      setPermModalVisible(false);
      setEditingRole(null);
      await loadRoles();
    }
    setSaving(false);
  };

  const handleCancelModal = () => {
    setPermModalVisible(false);
    setEditingRole(null);
    setSelectedPermissions([]);
  };

  // Permission groups for organized display in the modal
  const permissionGroups = [
    {
      label: 'Dashboard',
      permissions: ALL_PERMISSIONS.filter(p => p.startsWith('dashboard:')),
    },
    {
      label: 'User Management',
      permissions: ALL_PERMISSIONS.filter(p => p.startsWith('user:')),
    },
    {
      label: 'Role Management',
      permissions: ALL_PERMISSIONS.filter(p => p.startsWith('role:')),
    },
    {
      label: 'Menu Management',
      permissions: ALL_PERMISSIONS.filter(p => p.startsWith('menu:')),
    },
    {
      label: 'Security',
      permissions: ALL_PERMISSIONS.filter(p => p.startsWith('security:')),
    },
    {
      label: 'System',
      permissions: ALL_PERMISSIONS.filter(p => p.startsWith('system:') || p.startsWith('notification:')),
    },
  ];

  const columns: ColumnsType<Api.Role.RoleItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (text: string) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: 'Display Name',
      dataIndex: 'displayName',
      key: 'displayName',
      width: 180,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: 'Permissions',
      dataIndex: 'permissions',
      key: 'permissions',
      width: 300,
      render: (permissions: string[]) => (
        <Space size={[0, 4]} wrap>
          {permissions?.slice(0, 5).map(p => (
            <Tag key={p} color="processing">
              {p}
            </Tag>
          ))}
          {permissions && permissions.length > 5 && (
            <Tag>+{permissions.length - 5} more</Tag>
          )}
        </Space>
      ),
    },
    {
      title: 'Created At',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (text: string) => (text ? new Date(text).toLocaleString() : '-'),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 160,
      render: (_: unknown, record: Api.Role.RoleItem) => (
        <Space>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEditPermissions(record)}
          >
            Edit Permissions
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer title="Role Management">
      <Card>
        <Table<Api.Role.RoleItem>
          columns={columns}
          dataSource={roles}
          rowKey="id"
          loading={loading}
          pagination={false}
          size="middle"
        />
      </Card>

      {/* Permission Editing Modal */}
      <Modal
        title={`Edit Permissions - ${editingRole?.displayName || ''}`}
        open={permModalVisible}
        onOk={handleSavePermissions}
        onCancel={handleCancelModal}
        confirmLoading={saving}
        width={640}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          {permissionGroups.map(group => (
            <div key={group.label}>
              <div style={{ fontWeight: 600, marginBottom: 8 }}>{group.label}</div>
              <Checkbox.Group
                value={selectedPermissions}
                onChange={(values) => handlePermissionChange(values as string[])}
              >
                <Space wrap>
                  {group.permissions.map(perm => (
                    <Checkbox key={perm} value={perm}>
                      {perm.split(':')[1]}
                    </Checkbox>
                  ))}
                </Space>
              </Checkbox.Group>
            </div>
          ))}
        </Space>
      </Modal>
    </PageContainer>
  );
}
