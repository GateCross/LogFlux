/**
 * Menu Management Page (Task 14.1)
 *
 * Tree-structured menu table with add/edit/delete operations.
 * Fields: name, path, component, i18nKey, icon, roles, order, hideInMenu.
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
  InputNumber,
  Select,
  Switch,
  Popconfirm,
} from 'antd';
import { PageContainer } from '@ant-design/pro-components';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { request } from '@/utils/request';

// ---------------------------------------------------------------------------
// Inline service functions for menu management
// ---------------------------------------------------------------------------

interface MenuItem {
  id: number;
  name: string;
  path: string;
  component?: string;
  i18nKey?: string;
  icon?: string;
  roles?: string[];
  order?: number;
  hideInMenu?: boolean;
  parentId?: number;
  children?: MenuItem[];
  createdAt: string;
  updatedAt: string;
}

async function fetchMenuTree() {
  return request<{ list: MenuItem[] }>({ url: '/api/menu/tree' });
}

async function fetchCreateMenu(data: Omit<MenuItem, 'id' | 'createdAt' | 'updatedAt'>) {
  return request<void>({ url: '/api/menu', method: 'post', data });
}

async function fetchUpdateMenu(id: number, data: Partial<MenuItem>) {
  return request<void>({ url: `/api/menu/${id}`, method: 'put', data });
}

async function fetchDeleteMenu(id: number) {
  return request<void>({ url: `/api/menu/${id}`, method: 'delete' });
}

async function fetchRoleOptions() {
  return request<{ list: { name: string; displayName: string }[] }>({ url: '/api/role/list' });
}

// ---------------------------------------------------------------------------
// Helper: flatten tree to find parent options
// ---------------------------------------------------------------------------

function flattenMenuTree(tree: MenuItem[], depth = 0): { value: number; label: string; depth: number }[] {
  const result: { value: number; label: string; depth: number }[] = [];
  for (const item of tree) {
    result.push({ value: item.id, label: `${'  '.repeat(depth)}${item.name}`, depth });
    if (item.children && item.children.length > 0) {
      result.push(...flattenMenuTree(item.children, depth + 1));
    }
  }
  return result;
}

// ---------------------------------------------------------------------------
// Page component
// ---------------------------------------------------------------------------

export default function MenuManagementPage() {
  const [loading, setLoading] = useState(false);
  const [menuTree, setMenuTree] = useState<MenuItem[]>([]);
  const [roles, setRoles] = useState<{ name: string; displayName: string }[]>([]);

  // Modal state
  const [modalVisible, setModalVisible] = useState(false);
  const [editingMenu, setEditingMenu] = useState<MenuItem | null>(null);
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  const loadMenuTree = useCallback(async () => {
    setLoading(true);
    const { data, error } = await fetchMenuTree();
    if (error) {
      message.error(error.message || 'Failed to load menu tree');
    } else if (data) {
      setMenuTree(data.list || []);
    }
    setLoading(false);
  }, []);

  const loadRoles = useCallback(async () => {
    const { data } = await fetchRoleOptions();
    if (data) {
      setRoles(data.list || []);
    }
  }, []);

  useEffect(() => {
    loadMenuTree();
    loadRoles();
  }, [loadMenuTree, loadRoles]);

  const handleAdd = (parentId?: number) => {
    setEditingMenu(null);
    form.resetFields();
    if (parentId) {
      form.setFieldsValue({ parentId });
    }
    setModalVisible(true);
  };

  const handleEdit = (menu: MenuItem) => {
    setEditingMenu(menu);
    form.setFieldsValue({
      name: menu.name,
      path: menu.path,
      component: menu.component,
      i18nKey: menu.i18nKey,
      icon: menu.icon,
      roles: menu.roles,
      order: menu.order,
      hideInMenu: menu.hideInMenu,
      parentId: menu.parentId,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    const { error } = await fetchDeleteMenu(id);
    if (error) {
      message.error(error.message || 'Failed to delete menu');
    } else {
      message.success('Menu deleted successfully');
      loadMenuTree();
    }
  };

  const handleSaveMenu = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);

      if (editingMenu) {
        const { error } = await fetchUpdateMenu(editingMenu.id, values);
        if (error) {
          message.error(error.message || 'Failed to update menu');
        } else {
          message.success('Menu updated successfully');
          setModalVisible(false);
          loadMenuTree();
        }
      } else {
        const { error } = await fetchCreateMenu(values);
        if (error) {
          message.error(error.message || 'Failed to create menu');
        } else {
          message.success('Menu created successfully');
          setModalVisible(false);
          loadMenuTree();
        }
      }
    } catch {
      // Form validation failed
    }
    setSaving(false);
  };

  const handleCancelModal = () => {
    setModalVisible(false);
    setEditingMenu(null);
    form.resetFields();
  };

  const parentOptions = flattenMenuTree(menuTree).map(opt => ({
    label: opt.label,
    value: opt.value,
  }));

  const columns: ColumnsType<MenuItem> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (text: string) => <strong>{text}</strong>,
    },
    {
      title: 'Path',
      dataIndex: 'path',
      key: 'path',
      width: 200,
      render: (text: string) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: 'Component',
      dataIndex: 'component',
      key: 'component',
      width: 180,
      ellipsis: true,
    },
    {
      title: 'Icon',
      dataIndex: 'icon',
      key: 'icon',
      width: 120,
    },
    {
      title: 'i18n Key',
      dataIndex: 'i18nKey',
      key: 'i18nKey',
      width: 150,
    },
    {
      title: 'Roles',
      dataIndex: 'roles',
      key: 'roles',
      width: 200,
      render: (menuRoles: string[]) => (
        <Space size={[0, 4]} wrap>
          {menuRoles?.map(r => (
            <Tag key={r} color="processing">
              {r}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: 'Order',
      dataIndex: 'order',
      key: 'order',
      width: 80,
      sorter: (a: MenuItem, b: MenuItem) => (a.order || 0) - (b.order || 0),
    },
    {
      title: 'Hide',
      dataIndex: 'hideInMenu',
      key: 'hideInMenu',
      width: 80,
      render: (hide: boolean) => (
        <Tag color={hide ? 'red' : 'green'}>{hide ? 'Yes' : 'No'}</Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 220,
      render: (_: unknown, record: MenuItem) => (
        <Space>
          <Button type="link" size="small" icon={<PlusOutlined />} onClick={() => handleAdd(record.id)}>
            Add Child
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            Edit
          </Button>
          <Popconfirm
            title="Are you sure you want to delete this menu item?"
            description="Child items will also be deleted."
            onConfirm={() => handleDelete(record.id)}
            okText="Yes"
            cancelText="No"
          >
            <Button type="link" danger size="small" icon={<DeleteOutlined />}>
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer title="Menu Management">
      <Card>
        {/* Toolbar */}
        <Space style={{ marginBottom: 16 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => handleAdd()}>
            Add Root Menu
          </Button>
        </Space>

        {/* Tree table */}
        <Table<MenuItem>
          columns={columns}
          dataSource={menuTree}
          rowKey="id"
          loading={loading}
          pagination={false}
          scroll={{ x: 1400 }}
          expandable={{
            defaultExpandAllRows: true,
            childrenColumnName: 'children',
          }}
          size="middle"
        />
      </Card>

      {/* Add/Edit Menu Modal */}
      <Modal
        title={editingMenu ? 'Edit Menu' : 'Add Menu'}
        open={modalVisible}
        onOk={handleSaveMenu}
        onCancel={handleCancelModal}
        confirmLoading={saving}
        destroyOnClose
        width={600}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <Form.Item label="Parent Menu" name="parentId">
            <Select
              placeholder="Select parent menu (empty for root)"
              allowClear
              options={parentOptions}
            />
          </Form.Item>
          <Form.Item
            label="Name"
            name="name"
            rules={[{ required: true, message: 'Please enter menu name' }]}
          >
            <Input placeholder="Enter menu name" />
          </Form.Item>
          <Form.Item
            label="Path"
            name="path"
            rules={[{ required: true, message: 'Please enter route path' }]}
          >
            <Input placeholder="Enter route path (e.g. /manage/user)" />
          </Form.Item>
          <Form.Item label="Component" name="component">
            <Input placeholder="Enter component identifier (e.g. view.manage_user)" />
          </Form.Item>
          <Form.Item label="i18n Key" name="i18nKey">
            <Input placeholder="Enter i18n key (e.g. route.manage_user)" />
          </Form.Item>
          <Form.Item label="Icon" name="icon">
            <Input placeholder="Enter icon name (e.g. material-symbols:route)" />
          </Form.Item>
          <Form.Item label="Roles" name="roles">
            <Select
              mode="multiple"
              placeholder="Select allowed roles"
              options={roles.map(r => ({ label: r.displayName, value: r.name }))}
            />
          </Form.Item>
          <Form.Item label="Order" name="order" initialValue={0}>
            <InputNumber min={0} max={9999} style={{ width: '100%' }} placeholder="Sort order" />
          </Form.Item>
          <Form.Item label="Hide in Menu" name="hideInMenu" valuePropName="checked" initialValue={false}>
            <Switch checkedChildren="Yes" unCheckedChildren="No" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
