import { useState, useEffect } from 'react'
import { Card, Form, Input, Button, message, Typography, Divider, Space, Tag } from 'antd'
import { SaveOutlined, ReloadOutlined, ApiOutlined } from '@ant-design/icons'
import { healthApi } from '../services/api'

const { Title, Text, Paragraph } = Typography

interface SettingsForm {
  apiKey: string
  tenantId: string
  apiBaseUrl: string
}

export default function Settings() {
  const [form] = Form.useForm<SettingsForm>()
  const [serverStatus, setServerStatus] = useState<'online' | 'offline' | 'checking'>('checking')

  useEffect(() => {
    // 从localStorage加载设置
    const apiKey = localStorage.getItem('api_key') || 'test-api-key-12345'
    const tenantId = localStorage.getItem('tenant_id') || 'default'
    const apiBaseUrl = localStorage.getItem('api_base_url') || '/api/v1'

    form.setFieldsValue({ apiKey, tenantId, apiBaseUrl })
    checkServerStatus()
  }, [form])

  const checkServerStatus = async () => {
    setServerStatus('checking')
    try {
      await healthApi.check()
      setServerStatus('online')
    } catch {
      setServerStatus('offline')
    }
  }

  const handleSave = (values: SettingsForm) => {
    localStorage.setItem('api_key', values.apiKey)
    localStorage.setItem('tenant_id', values.tenantId)
    localStorage.setItem('api_base_url', values.apiBaseUrl)
    message.success('设置已保存')
  }

  const handleReset = () => {
    form.setFieldsValue({
      apiKey: 'test-api-key-12345',
      tenantId: 'default',
      apiBaseUrl: '/api/v1',
    })
    message.info('已重置为默认值')
  }

  return (
    <div className="space-y-6">
      <Title level={4}>系统设置</Title>

      <Card title="服务状态">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <ApiOutlined className="text-2xl text-gray-400" />
            <div>
              <Text strong>AI-Trace Server</Text>
              <br />
              <Text type="secondary">后端服务连接状态</Text>
            </div>
          </div>
          <Space>
            <Tag
              color={
                serverStatus === 'online'
                  ? 'success'
                  : serverStatus === 'offline'
                  ? 'error'
                  : 'processing'
              }
            >
              {serverStatus === 'online'
                ? '在线'
                : serverStatus === 'offline'
                ? '离线'
                : '检查中...'}
            </Tag>
            <Button icon={<ReloadOutlined />} onClick={checkServerStatus} size="small">
              刷新
            </Button>
          </Space>
        </div>
      </Card>

      <Card title="API 配置">
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={{
            apiKey: 'test-api-key-12345',
            tenantId: 'default',
            apiBaseUrl: '/api/v1',
          }}
        >
          <Form.Item
            name="apiKey"
            label="API Key"
            rules={[{ required: true, message: '请输入API Key' }]}
          >
            <Input.Password placeholder="输入您的API Key" />
          </Form.Item>

          <Form.Item
            name="tenantId"
            label="租户ID"
            rules={[{ required: true, message: '请输入租户ID' }]}
          >
            <Input placeholder="输入租户ID" />
          </Form.Item>

          <Form.Item name="apiBaseUrl" label="API Base URL">
            <Input placeholder="/api/v1" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<SaveOutlined />}>
                保存设置
              </Button>
              <Button onClick={handleReset}>重置默认</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card title="关于">
        <Paragraph>
          <Text strong>AI-Trace Console</Text> v0.1.0
        </Paragraph>
        <Paragraph type="secondary">
          企业级AI全链路审计系统控制台，提供事件追踪、存证管理、在线验证等功能。
        </Paragraph>
        <Divider />
        <div className="space-y-2">
          <div>
            <Text type="secondary">技术栈：</Text>
            <Space className="ml-2">
              <Tag>React 18</Tag>
              <Tag>TypeScript</Tag>
              <Tag>Ant Design 5</Tag>
              <Tag>TailwindCSS</Tag>
              <Tag>Vite</Tag>
            </Space>
          </div>
          <div>
            <Text type="secondary">后端：</Text>
            <Space className="ml-2">
              <Tag>Go</Tag>
              <Tag>Gin</Tag>
              <Tag>PostgreSQL</Tag>
              <Tag>Redis</Tag>
              <Tag>MinIO</Tag>
            </Space>
          </div>
        </div>
        <Divider />
        <Paragraph type="secondary">
          开源项目：
          <a
            href="https://github.com/ai-trace/ai-trace"
            target="_blank"
            rel="noopener noreferrer"
            className="ml-2"
          >
            GitHub
          </a>
        </Paragraph>
      </Card>
    </div>
  )
}
