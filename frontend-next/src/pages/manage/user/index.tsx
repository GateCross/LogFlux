/**
 * User Management Page (Task 14.1)
 *
 * Paginated user table with search, add/edit/delete users via modal forms,
 * toggle enable/disable, and role assignment from API-fetched roles.
 */
import { useEffect, useState, useCallback } from 'react';
import {
  Table,
  Button,
  Modal,
  Space,
  message,
  Card,
  Tag,
  Form,
  Input,
  Select,
  Switch,
  Popconfirm,
} from 'antd';
import { PageContainer } from '@ant-design/pro-components';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useIntl } from '@umijs/max';
import type { ColumnsType } from 'antd/es/table';
import { request } from '@/utils/request';

// ---------------------------------------------------------------------------
// Inline service functions for user management
// ---------------------------------------------------------------------------

interface UserItem {
  id: number;
  username: string;
  displayName: string;
  email: string;
  phone: string;
  roles: string[];
  status: '1' | '2';
  createdAt: string;
  updatedAt: string;
}

async function fetchUserList(params: { page: number; pageSize: number; username?: string }) {
  return request<{ list: UserItem[]; total: number }>({
    url: '/api/user/list',
    params: { current: params.page, size: params.pageSize, username: params.username },
  });
}

async function fetchCreateUser(data: {
  username: string;
  displayName: string;
  email: string;
  phone: string;
  roles: string[];
  password?: string;
}) {
  return request<void>({ url: '/api/user', method: 'post', data });
}

async function fetchUpdateUser(id: number, data: Partial<UserItem & { password?: string }>) {
  return request<void>({ url: `/api/user/${id}`, method: 'put', data });
}

async function fetchDeleteUser(id: number) {
  return request<void>({ url: `/api/user/${id}`, method: 'delete' });
}

async function fetchToggleUserStatus(id: number, status: '1' | '2') {
  return request<void>({ url: `/api/user/${id}/status`, method: 'put', data: { status } });
}

async function fetchRoleOptions() {
  return request<{ list: { name: string; displayName: string }[] }>({ url: '/api/role/list' });
}

// ---------------------------------------------------------------------------
// Page component
// ---------------------------------------------------------------------------

export default function UserManagementPage() {
  const intl = useIntl();
  const t = useCallback(
    (id: string, fb?: string) => intl.formatMessage({ id, defaultMessage: fb }),
    [intl],
  );

  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [searchUsername, setSearchUsername] = useState('');
  const [roles, setRoles] = useState<{ name: string; displayName: string }[]>([]);

  // Modal state
  const [modalVisible, setModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<UserItem | null>(null);
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  const loadUsers = useCallback(async (p: number, ps: number, username?: string) => {
    setLoading(true);
    const { data, error } = await fetchUserList({ page: p, pageSize: ps, username });
    if (error) {
      message.error(error.message || t('common.error', 'Error'));
    } else if (data) {
      setUsers(data.list || []);
      setTotal(data.total || 0);
    }
    setLoading(false);
  }, [t]);

  const loadRoles = useCallback(async () => {
    const { data } = await fetchRoleOptions();
    if (data) {
      setRoles(data.list || []);
    }
  }, []);

  useEffect(() => {
    loadUsers(page, pageSize, searchUsername);
    loadRoles();
  }, [page, pageSize, loadUsers, loadRoles]);

  const handleSearch = () => {
    setPage(1);
    loadUsers(1, pageSize, searchUsername);
  };

  const handleReset = () => {
    setSearchUsername('');
    setPage(1);
    loadUsers(1, pageSize, '');
  };

  const handleAdd = () => {
    setEditingUser(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (user: UserItem) => {
    setEditingUser(user);
    form.setFieldsValue({
      username: user.username,
      displayName: user.displayName,
      email: user.email,
      phone: user.phone,
      roles: user.roles,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    const { error } = await fetchDeleteUser(id);
    if (error) {
      message.error(error.message || t('common.deleteFailed', 'Delete Failed'));
    } else {
      message.success(t('common.deleteSuccess', 'Delete Success'));
      loadUsers(page, pageSize, searchUsername);
    }
  };

  const handleToggleStatus = async (user: UserItem) => {
    const newStatus: '1' | '2' = user.status === '1' ? '2' : '1';
    const { error } = await fetchToggleUserStatus(user.id, newStatus);
    if (error) {
      message.error(error.message || t('common.updateFailed', 'Update Failed'));
    } else {
      message.success(
        newStatus === '1'
          ? t('manage.user.enableSuccess', 'User enabled successfully')
          : t('manage.user.disableSuccess', 'User disabled successfully'),
      );
      loadUsers(page, pageSize, searchUsername);
    }
  };

  const handleSaveUser = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);

      if (editingUser) {
        const { error } = await fetchUpdateUser(editingUser.id, values);
        if (error) {
          message.error(error.message || t('common.updateFailed', 'Update Failed'));
        } else {
          message.success(t('common.updateSuccess', 'Update Success'));
          setModalVisible(false);
          loadUsers(page, pageSize, searchUsername);
        }
      } else {
        const { error } = await fetchCreateUser(values);
        if (error) {
          message.error(error.message || t('common.addFailed', 'Add Failed'));
        } else {
          message.success(t('common.addSuccess', 'Add Success'));
          setModalVisible(false);
          setPage(1);
          loadUsers(1, pageSize, searchUsername);
        }
      }
    } catch {
      // Form validation failed
    }
    setSaving(false);
  };

  const handleCancelModal = () => {
    setModalVisible(false);
    setEditingUser(null);
    form.resetFields();
  };

  const columns: ColumnsType<UserItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: t('manage.user.username', 'Username'),
      dataIndex: 'username',
      key: 'username',
      width: 140,
    },
    {
      title: t('manage.user.displayName', 'Display Name'),
      dataIndex: 'displayName',
      key: 'displayName',
      width: 150,
    },
    {
      title: t('manage.user.email', 'Email'),
      dataIndex: 'email',
      key: 'email',
      width: 200,
    },
    {
      title: t('manage.user.phone', 'Phone'),
      dataIndex: 'phone',
      key: 'phone',
      width: 140,
    },
    {
      title: t('manage.user.roles', 'Roles'),
      dataIndex: 'roles',
      key: 'roles',
      width: 200,
      render: (userRoles: string[]) => (
        <Space size={[0, 4]} wrap>
          {userRoles?.map(r => (
            <Tag key={r} color="blue">
              {r}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: t('manage.user.status', 'Status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: '1' | '2', record: UserItem) => (
        <Switch
          checked={status === '1'}
          checkedChildren={t('common.enabled', 'Enabled')}
          unCheckedChildren={t('common.disabled', 'Disabled')}
          onChange={() => handleToggleStatus(record)}
        />
      ),
    },
    {
      title: t('common.createdAt', 'Created At'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (text: string) => (text ? new Date(text).toLocaleString() : '-'),
    },
    {
      title: t('common.action', 'Actions'),
      key: 'actions',
      width: 160,
      fixed: 'right',
      render: (_: unknown, record: UserItem) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            {t('common.edit', 'Edit')}
          </Button>
          <Popconfirm
            title={t('manage.user.confirmDeleteUser', 'Are you sure you want to delete this user?')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.yesOrNo.yes', 'Yes')}
            cancelText={t('common.yesOrNo.no', 'No')}
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              {t('common.delete', 'Delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer title={t('manage.user.title', 'User Management')}>
      <Card>
        {/* Search bar */}
        <Space style={{ marginBottom: 16 }}>
          <Input
            placeholder={t('manage.user.searchPlaceholder', 'Search by username')}
            value={searchUsername}
            onChange={(e) => setSearchUsername(e.target.value)}
            onPressEnter={handleSearch}
            style={{ width: 240 }}
            allowClear
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            {t('common.search', 'Search')}
          </Button>
          <Button onClick={handleReset}>{t('common.reset', 'Reset')}</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            {t('manage.user.addUser', 'Add User')}
          </Button>
        </Space>

        {/* User table */}
        <Table<UserItem>
          columns={columns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) =>
              intl.formatMessage(
                { id: 'manage.user.totalUsers', defaultMessage: 'Total {total} users' },
                { total },
              ),
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      {/* Add/Edit User Modal */}
      <Modal
        title={
          editingUser
            ? t('manage.user.editUser', 'Edit User')
            : t('manage.user.addUser', 'Add User')
        }
        open={modalVisible}
        onOk={handleSaveUser}
        onCancel={handleCancelModal}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          autoComplete="off"
        >
          <Form.Item
            label={t('manage.user.username', 'Username')}
            name="username"
            rules={[{ required: true, message: t('manage.user.enterUsername', 'Please enter username') }]}
          >
            <Input
              disabled={!!editingUser}
              placeholder={t('manage.user.usernamePlaceholder', 'Enter username')}
            />
          </Form.Item>
          <Form.Item
            label={t('manage.user.displayName', 'Display Name')}
            name="displayName"
            rules={[{ required: true, message: t('manage.user.enterDisplayName', 'Please enter display name') }]}
          >
            <Input placeholder={t('manage.user.displayNamePlaceholder', 'Enter display name')} />
          </Form.Item>
          <Form.Item
            label={t('manage.user.email', 'Email')}
            name="email"
            rules={[
              { required: true, message: t('manage.user.enterEmail', 'Please enter email') },
              { type: 'email', message: t('manage.user.invalidEmail', 'Invalid email format') },
            ]}
          >
            <Input placeholder={t('manage.user.emailPlaceholder', 'Enter email')} />
          </Form.Item>
          <Form.Item
            label={t('manage.user.phone', 'Phone')}
            name="phone"
          >
            <Input placeholder={t('manage.user.phonePlaceholder', 'Enter phone number')} />
          </Form.Item>
          <Form.Item
            label={t('manage.user.roles', 'Roles')}
            name="roles"
            rules={[{ required: true, message: t('manage.user.selectRoleRequired', 'Please select at least one role') }]}
          >
            <Select
              mode="multiple"
              placeholder={t('manage.user.selectRoles', 'Select roles')}
              options={roles.map(r => ({ label: r.displayName, value: r.name }))}
            />
          </Form.Item>
          {!editingUser && (
            <Form.Item
              label={t('manage.user.password', 'Password')}
              name="password"
              rules={[
                { required: true, message: t('manage.user.enterPassword', 'Please enter password') },
                { min: 6, message: t('manage.user.passwordMinLength', 'Password must be at least 6 characters') },
              ]}
            >
              <Input.Password placeholder={t('manage.user.passwordPlaceholder', 'Enter password')} />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </PageContainer>
  );
}
