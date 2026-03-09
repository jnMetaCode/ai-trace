import { useEffect, useState } from 'react'
import { Card, Row, Col, Statistic, Table, Tag, Typography } from 'antd'
import {
  FileSearchOutlined,
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { eventApi, certApi } from '../services/api'

const { Title } = Typography

interface DashboardStats {
  totalEvents: number
  totalCerts: number
  verifiedCerts: number
  recentEvents: unknown[]
  recentCerts: unknown[]
}

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats>({
    totalEvents: 0,
    totalCerts: 0,
    verifiedCerts: 0,
    recentEvents: [],
    recentCerts: [],
  })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadDashboardData()
  }, [])

  const loadDashboardData = async () => {
    try {
      setLoading(true)
      const [eventsRes, certsRes] = await Promise.all([
        eventApi.search({ page_size: 10 }).catch(() => ({ events: [], size: 0 })),
        certApi.search({ page_size: 10 }).catch(() => ({ certificates: [], size: 0 })),
      ])

      setStats({
        totalEvents: (eventsRes as { size?: number }).size || 0,
        totalCerts: (certsRes as { size?: number }).size || 0,
        verifiedCerts: (certsRes as { size?: number }).size || 0,
        recentEvents: (eventsRes as { events?: unknown[] }).events || [],
        recentCerts: (certsRes as { certificates?: unknown[] }).certificates || [],
      })
    } catch (error) {
      console.error('Failed to load dashboard data:', error)
    } finally {
      setLoading(false)
    }
  }

  const eventTypeOption = {
    tooltip: { trigger: 'item' },
    legend: { orient: 'vertical', left: 'left' },
    series: [
      {
        name: '事件类型',
        type: 'pie',
        radius: '70%',
        data: [
          { value: 35, name: 'INPUT' },
          { value: 30, name: 'MODEL' },
          { value: 20, name: 'OUTPUT' },
          { value: 10, name: 'RETRIEVAL' },
          { value: 5, name: 'TOOL_CALL' },
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
      },
    ],
  }

  const trendOption = {
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
    },
    yAxis: { type: 'value' },
    series: [
      {
        name: '事件数',
        type: 'line',
        smooth: true,
        data: [120, 200, 150, 80, 70, 110, 130],
        areaStyle: {
          opacity: 0.3,
        },
      },
      {
        name: '存证数',
        type: 'line',
        smooth: true,
        data: [20, 35, 25, 15, 12, 18, 22],
        areaStyle: {
          opacity: 0.3,
        },
      },
    ],
  }

  const eventColumns = [
    {
      title: '事件ID',
      dataIndex: 'event_id',
      key: 'event_id',
      ellipsis: true,
      width: 150,
    },
    {
      title: '类型',
      dataIndex: 'event_type',
      key: 'event_type',
      render: (type: string) => {
        const colors: Record<string, string> = {
          INPUT: 'blue',
          MODEL: 'purple',
          OUTPUT: 'green',
          RETRIEVAL: 'orange',
          TOOL_CALL: 'cyan',
        }
        return <Tag color={colors[type] || 'default'}>{type}</Tag>
      },
    },
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      render: (ts: string) => new Date(ts).toLocaleString(),
    },
  ]

  const certColumns = [
    {
      title: '存证ID',
      dataIndex: 'cert_id',
      key: 'cert_id',
      ellipsis: true,
      width: 150,
    },
    {
      title: '级别',
      dataIndex: 'evidence_level',
      key: 'evidence_level',
      render: (level: string) => (
        <Tag color={level === 'L3' ? 'gold' : level === 'L2' ? 'blue' : 'default'}>
          {level}
        </Tag>
      ),
    },
    {
      title: '事件数',
      dataIndex: 'event_count',
      key: 'event_count',
    },
  ]

  return (
    <div className="space-y-6">
      <Title level={4}>系统概览</Title>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总事件数"
              value={stats.totalEvents}
              prefix={<FileSearchOutlined className="text-blue-500" />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="存证数量"
              value={stats.totalCerts}
              prefix={<SafetyCertificateOutlined className="text-purple-500" />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="已验证"
              value={stats.verifiedCerts}
              prefix={<CheckCircleOutlined className="text-green-500" />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="今日新增"
              value={0}
              prefix={<ClockCircleOutlined className="text-orange-500" />}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="事件类型分布">
            <ReactECharts option={eventTypeOption} style={{ height: 300 }} />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="近7天趋势">
            <ReactECharts option={trendOption} style={{ height: 300 }} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="最近事件">
            <Table
              columns={eventColumns}
              dataSource={stats.recentEvents as Record<string, unknown>[]}
              rowKey="event_id"
              size="small"
              pagination={false}
              loading={loading}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="最近存证">
            <Table
              columns={certColumns}
              dataSource={stats.recentCerts as Record<string, unknown>[]}
              rowKey="cert_id"
              size="small"
              pagination={false}
              loading={loading}
            />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
