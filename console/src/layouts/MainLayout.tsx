import { useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, theme, Avatar, Dropdown, Space } from 'antd'
import {
  DashboardOutlined,
  FileSearchOutlined,
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  SettingOutlined,
  UserOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons'
import type { MenuProps } from 'antd'

const { Header, Sider, Content } = Layout

const menuItems: MenuProps['items'] = [
  {
    key: '/dashboard',
    icon: <DashboardOutlined />,
    label: '仪表盘',
  },
  {
    key: '/events',
    icon: <FileSearchOutlined />,
    label: '事件追踪',
  },
  {
    key: '/certificates',
    icon: <SafetyCertificateOutlined />,
    label: '存证管理',
  },
  {
    key: '/verify',
    icon: <CheckCircleOutlined />,
    label: '在线验证',
  },
  {
    key: '/settings',
    icon: <SettingOutlined />,
    label: '系统设置',
  },
]

const userMenuItems: MenuProps['items'] = [
  { key: 'profile', label: '个人设置' },
  { key: 'logout', label: '退出登录' },
]

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken()

  const handleMenuClick: MenuProps['onClick'] = (e) => {
    navigate(e.key)
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        theme="light"
        style={{
          boxShadow: '2px 0 8px rgba(0,0,0,0.05)',
        }}
      >
        <div className="flex items-center justify-center h-16 border-b border-gray-100">
          <div className="flex items-center gap-2">
            <svg width="32" height="32" viewBox="0 0 100 100">
              <defs>
                <linearGradient id="logoGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" style={{ stopColor: '#1890ff', stopOpacity: 1 }} />
                  <stop offset="100%" style={{ stopColor: '#722ed1', stopOpacity: 1 }} />
                </linearGradient>
              </defs>
              <rect x="10" y="10" width="80" height="80" rx="15" fill="url(#logoGrad)" />
              <text x="50" y="62" fontFamily="Arial" fontSize="36" fontWeight="bold" fill="white" textAnchor="middle">
                AT
              </text>
            </svg>
            {!collapsed && <span className="text-lg font-semibold text-gray-800">AI-Trace</span>}
          </div>
        </div>
        <Menu
          theme="light"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ borderRight: 0 }}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 24px',
            background: colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            boxShadow: '0 1px 4px rgba(0,0,0,0.05)',
          }}
        >
          <div
            onClick={() => setCollapsed(!collapsed)}
            className="cursor-pointer text-lg text-gray-600 hover:text-blue-500 transition-colors"
          >
            {collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          </div>
          <Space size="middle">
            <span className="text-gray-500 text-sm">租户: Default</span>
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Space className="cursor-pointer">
                <Avatar size="small" icon={<UserOutlined />} />
                <span className="text-gray-700">Admin</span>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content
          style={{
            margin: '24px',
            padding: 24,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
            minHeight: 280,
            overflow: 'auto',
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
