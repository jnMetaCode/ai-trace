import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, Table, Tag, Button, Space, message, Typography, Modal, Input } from 'antd'
import {
  ReloadOutlined,
  EyeOutlined,
  CheckCircleOutlined,
  PlusOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { certApi, Certificate } from '../services/api'
import dayjs from 'dayjs'

const { Title } = Typography

export default function Certificates() {
  const navigate = useNavigate()
  const [certs, setCerts] = useState<Certificate[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [commitModal, setCommitModal] = useState(false)
  const [traceIdInput, setTraceIdInput] = useState('')
  const [committing, setCommitting] = useState(false)

  useEffect(() => {
    loadCerts()
  }, [pagination.current, pagination.pageSize])

  const loadCerts = async () => {
    setLoading(true)
    try {
      const data = await certApi.search({
        page: pagination.current,
        page_size: pagination.pageSize,
      })
      setCerts(data.certificates || [])
      setPagination((prev) => ({ ...prev, total: data.size || 0 }))
    } catch (error) {
      console.error('Failed to load certificates:', error)
      message.error('加载存证列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCommit = async () => {
    if (!traceIdInput.trim()) {
      message.warning('请输入 Trace ID')
      return
    }

    setCommitting(true)
    try {
      const data = await certApi.commit(traceIdInput.trim())
      message.success(`存证创建成功: ${data.cert_id}`)
      setCommitModal(false)
      setTraceIdInput('')
      loadCerts()
    } catch (error) {
      console.error('Failed to commit:', error)
      message.error('存证创建失败')
    } finally {
      setCommitting(false)
    }
  }

  const handleVerify = async (certId: string) => {
    try {
      const data = await certApi.verify(certId)
      if (data.valid) {
        message.success('验证通过：存证有效')
      } else {
        message.error('验证失败：存证可能已被篡改')
      }
    } catch (error) {
      console.error('Failed to verify:', error)
      message.error('验证失败')
    }
  }

  const columns: ColumnsType<Certificate> = [
    {
      title: '存证ID',
      dataIndex: 'cert_id',
      key: 'cert_id',
      width: 180,
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
      title: '存证级别',
      dataIndex: 'evidence_level',
      key: 'evidence_level',
      width: 100,
      render: (level: string) => {
        const colors: Record<string, string> = {
          L1: 'default',
          L2: 'blue',
          L3: 'gold',
        }
        const labels: Record<string, string> = {
          L1: 'L1 本地',
          L2: 'L2 WORM',
          L3: 'L3 区块链',
        }
        return <Tag color={colors[level]}>{labels[level] || level}</Tag>
      },
    },
    {
      title: '事件数',
      dataIndex: 'event_count',
      key: 'event_count',
      width: 80,
    },
    {
      title: 'Root Hash',
      dataIndex: 'root_hash',
      key: 'root_hash',
      ellipsis: true,
      render: (hash: string) => (
        <span className="font-mono text-xs text-gray-400" title={hash}>
          {hash?.substring(0, 24)}...
        </span>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (ts: string) => dayjs(ts).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => navigate(`/certificates/${record.cert_id}`)}
          >
            详情
          </Button>
          <Button
            type="link"
            size="small"
            icon={<CheckCircleOutlined />}
            onClick={() => handleVerify(record.cert_id)}
          >
            验证
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Title level={4} style={{ margin: 0 }}>
          存证管理
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadCerts}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCommitModal(true)}>
            新建存证
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={certs}
          rowKey="cert_id"
          loading={loading}
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) =>
              setPagination((prev) => ({ ...prev, current: page, pageSize })),
          }}
          scroll={{ x: 1000 }}
        />
      </Card>

      <Modal
        title="创建新存证"
        open={commitModal}
        onOk={handleCommit}
        onCancel={() => {
          setCommitModal(false)
          setTraceIdInput('')
        }}
        confirmLoading={committing}
      >
        <div className="py-4">
          <p className="mb-2 text-gray-600">请输入要生成存证的 Trace ID：</p>
          <Input
            placeholder="trc_xxxxxxxx"
            value={traceIdInput}
            onChange={(e) => setTraceIdInput(e.target.value)}
          />
        </div>
      </Modal>
    </div>
  )
}
