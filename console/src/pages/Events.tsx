import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Tag, Input, Select, Button, Space, DatePicker, message, Typography } from 'antd'
import { SearchOutlined, ReloadOutlined, EyeOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { eventApi, Event } from '../services/api'
import dayjs from 'dayjs'

const { Title } = Typography
const { RangePicker } = DatePicker

const eventTypeOptions = [
  { value: '', label: '全部类型' },
  { value: 'INPUT', label: 'INPUT' },
  { value: 'MODEL', label: 'MODEL' },
  { value: 'OUTPUT', label: 'OUTPUT' },
  { value: 'RETRIEVAL', label: 'RETRIEVAL' },
  { value: 'TOOL_CALL', label: 'TOOL_CALL' },
  { value: 'POST_EDIT', label: 'POST_EDIT' },
]

export default function Events() {
  const navigate = useNavigate()
  const [events, setEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [filters, setFilters] = useState({
    trace_id: '',
    event_type: '',
    dateRange: null as [dayjs.Dayjs, dayjs.Dayjs] | null,
  })

  useEffect(() => {
    loadEvents()
  }, [pagination.current, pagination.pageSize])

  const loadEvents = async () => {
    setLoading(true)
    try {
      const params: Record<string, unknown> = {
        page: pagination.current,
        page_size: pagination.pageSize,
      }

      if (filters.trace_id) params.trace_id = filters.trace_id
      if (filters.event_type) params.event_type = filters.event_type
      if (filters.dateRange) {
        params.start_time = filters.dateRange[0].toISOString()
        params.end_time = filters.dateRange[1].toISOString()
      }

      const data = await eventApi.search(params as Parameters<typeof eventApi.search>[0])
      setEvents(data.events || [])
      setPagination((prev) => ({ ...prev, total: data.size || 0 }))
    } catch (error) {
      console.error('Failed to load events:', error)
      message.error('加载事件失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = () => {
    setPagination((prev) => ({ ...prev, current: 1 }))
    loadEvents()
  }

  const handleReset = () => {
    setFilters({ trace_id: '', event_type: '', dateRange: null })
    setPagination((prev) => ({ ...prev, current: 1 }))
  }

  const columns: ColumnsType<Event> = [
    {
      title: '事件ID',
      dataIndex: 'event_id',
      key: 'event_id',
      width: 180,
      ellipsis: true,
      render: (id: string) => (
        <span className="font-mono text-sm text-blue-600">{id}</span>
      ),
    },
    {
      title: 'Trace ID',
      dataIndex: 'trace_id',
      key: 'trace_id',
      width: 150,
      ellipsis: true,
      render: (id: string) => (
        <span className="font-mono text-xs text-gray-500">{id}</span>
      ),
    },
    {
      title: '类型',
      dataIndex: 'event_type',
      key: 'event_type',
      width: 120,
      render: (type: string) => {
        const colors: Record<string, string> = {
          INPUT: 'blue',
          MODEL: 'purple',
          OUTPUT: 'green',
          RETRIEVAL: 'orange',
          TOOL_CALL: 'cyan',
          POST_EDIT: 'magenta',
        }
        return <Tag color={colors[type] || 'default'}>{type}</Tag>
      },
    },
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (ts: string) => dayjs(ts).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '事件哈希',
      dataIndex: 'event_hash',
      key: 'event_hash',
      ellipsis: true,
      render: (hash: string) => (
        <span className="font-mono text-xs text-gray-400" title={hash}>
          {hash?.substring(0, 20)}...
        </span>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => navigate(`/events/${record.event_id}`)}
        >
          详情
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <Title level={4}>事件追踪</Title>

      <Card size="small">
        <Space wrap className="w-full">
          <Input
            placeholder="Trace ID"
            value={filters.trace_id}
            onChange={(e) => setFilters((prev) => ({ ...prev, trace_id: e.target.value }))}
            style={{ width: 200 }}
            allowClear
          />
          <Select
            value={filters.event_type}
            onChange={(value) => setFilters((prev) => ({ ...prev, event_type: value }))}
            options={eventTypeOptions}
            style={{ width: 140 }}
          />
          <RangePicker
            value={filters.dateRange}
            onChange={(dates) =>
              setFilters((prev) => ({
                ...prev,
                dateRange: dates as [dayjs.Dayjs, dayjs.Dayjs] | null,
              }))
            }
            showTime
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
            搜索
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset}>
            重置
          </Button>
        </Space>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={events}
          rowKey="event_id"
          loading={loading}
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) =>
              setPagination((prev) => ({ ...prev, current: page, pageSize })),
          }}
          scroll={{ x: 900 }}
        />
      </Card>
    </div>
  )
}
