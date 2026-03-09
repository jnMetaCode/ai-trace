import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Descriptions, Tag, Button, Spin, message, Typography, Space } from 'antd'
import { ArrowLeftOutlined, CopyOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { eventApi, certApi, EventDetail as EventDetailType } from '../services/api'
import dayjs from 'dayjs'

const { Title, Text } = Typography

export default function EventDetail() {
  const { eventId } = useParams<{ eventId: string }>()
  const navigate = useNavigate()
  const [event, setEvent] = useState<EventDetailType | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (eventId) {
      loadEvent(eventId)
    }
  }, [eventId])

  const loadEvent = async (id: string) => {
    try {
      setLoading(true)
      const data = await eventApi.get(id)
      setEvent(data)
    } catch (error) {
      console.error('Failed to load event:', error)
      message.error('加载事件详情失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text)
    message.success('已复制到剪贴板')
  }

  const handleCommitCert = async () => {
    if (!event) return
    try {
      const data = await certApi.commit(event.trace_id)
      message.success(`存证创建成功: ${data.cert_id}`)
      navigate(`/certificates/${data.cert_id}`)
    } catch (error) {
      console.error('Failed to commit cert:', error)
      message.error('存证创建失败')
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <Spin size="large" />
      </div>
    )
  }

  if (!event) {
    return (
      <div className="text-center py-12">
        <Text type="secondary">事件不存在</Text>
      </div>
    )
  }

  const typeColors: Record<string, string> = {
    INPUT: 'blue',
    MODEL: 'purple',
    OUTPUT: 'green',
    RETRIEVAL: 'orange',
    TOOL_CALL: 'cyan',
    POST_EDIT: 'magenta',
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)}>
            返回
          </Button>
          <Title level={4} style={{ margin: 0 }}>
            事件详情
          </Title>
        </Space>
        <Button
          type="primary"
          icon={<SafetyCertificateOutlined />}
          onClick={handleCommitCert}
        >
          生成存证
        </Button>
      </div>

      <Card title="基本信息">
        <Descriptions column={{ xs: 1, sm: 2, md: 2 }} bordered size="small">
          <Descriptions.Item label="事件ID">
            <Space>
              <Text className="font-mono text-sm">{event.event_id}</Text>
              <CopyOutlined
                className="text-gray-400 cursor-pointer hover:text-blue-500"
                onClick={() => handleCopy(event.event_id)}
              />
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="Trace ID">
            <Space>
              <Text className="font-mono text-sm">{event.trace_id}</Text>
              <CopyOutlined
                className="text-gray-400 cursor-pointer hover:text-blue-500"
                onClick={() => handleCopy(event.trace_id)}
              />
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="事件类型">
            <Tag color={typeColors[event.event_type] || 'default'}>{event.event_type}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="序列号">{event.sequence}</Descriptions.Item>
          <Descriptions.Item label="时间">
            {dayjs(event.timestamp).format('YYYY-MM-DD HH:mm:ss.SSS')}
          </Descriptions.Item>
          <Descriptions.Item label="事件哈希">
            <Space>
              <Text className="font-mono text-xs" style={{ wordBreak: 'break-all' }}>
                {event.event_hash}
              </Text>
              <CopyOutlined
                className="text-gray-400 cursor-pointer hover:text-blue-500"
                onClick={() => handleCopy(event.event_hash)}
              />
            </Space>
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="Payload 详情">
        <pre className="bg-gray-50 p-4 rounded-lg overflow-auto text-sm max-h-96">
          {JSON.stringify(event.payload, null, 2)}
        </pre>
      </Card>
    </div>
  )
}
