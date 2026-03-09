import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Spin,
  message,
  Typography,
  Space,
  Alert,
  Table,
  Modal,
} from 'antd'
import {
  ArrowLeftOutlined,
  CopyOutlined,
  CheckCircleOutlined,
  DownloadOutlined,
  ShareAltOutlined,
} from '@ant-design/icons'
import { certApi, CertificateDetail, VerifyResult } from '../services/api'
import dayjs from 'dayjs'

const { Title, Text } = Typography

export default function CertDetail() {
  const { certId } = useParams<{ certId: string }>()
  const navigate = useNavigate()
  const [cert, setCert] = useState<CertificateDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null)
  const [verifying, setVerifying] = useState(false)
  const [proofModal, setProofModal] = useState(false)
  const [proof, setProof] = useState<unknown>(null)

  useEffect(() => {
    if (certId) {
      loadCert(certId)
    }
  }, [certId])

  const loadCert = async (id: string) => {
    try {
      setLoading(true)
      const data = await certApi.get(id)
      setCert(data)
    } catch (error) {
      console.error('Failed to load certificate:', error)
      message.error('加载存证详情失败')
    } finally {
      setLoading(false)
    }
  }

  const handleVerify = async () => {
    if (!certId) return
    setVerifying(true)
    try {
      const data = await certApi.verify(certId)
      setVerifyResult(data)
      if (data.valid) {
        message.success('验证通过')
      } else {
        message.warning('验证未通过')
      }
    } catch (error) {
      console.error('Failed to verify:', error)
      message.error('验证失败')
    } finally {
      setVerifying(false)
    }
  }

  const handleGenerateProof = async () => {
    if (!certId || !cert) return
    try {
      const indices = cert.event_hashes.map((_, i) => i)
      const res = await certApi.generateProof(certId, indices, [])
      setProof(res)
      setProofModal(true)
    } catch (error) {
      console.error('Failed to generate proof:', error)
      message.error('生成证明失败')
    }
  }

  const handleDownloadProof = () => {
    if (!proof) return
    const blob = new Blob([JSON.stringify(proof, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `proof_${certId}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text)
    message.success('已复制到剪贴板')
  }

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <Spin size="large" />
      </div>
    )
  }

  if (!cert) {
    return (
      <div className="text-center py-12">
        <Text type="secondary">存证不存在</Text>
      </div>
    )
  }

  const levelColors: Record<string, string> = {
    L1: 'default',
    L2: 'blue',
    L3: 'gold',
  }

  const eventHashColumns = [
    { title: '序号', dataIndex: 'index', key: 'index', width: 60 },
    {
      title: '事件哈希',
      dataIndex: 'hash',
      key: 'hash',
      render: (hash: string) => (
        <span className="font-mono text-xs">{hash}</span>
      ),
    },
  ]

  const eventHashData = cert.event_hashes.map((hash, index) => ({
    key: index,
    index: index + 1,
    hash,
  }))

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)}>
            返回
          </Button>
          <Title level={4} style={{ margin: 0 }}>
            存证详情
          </Title>
        </Space>
        <Space>
          <Button icon={<ShareAltOutlined />} onClick={handleGenerateProof}>
            生成证明
          </Button>
          <Button
            type="primary"
            icon={<CheckCircleOutlined />}
            onClick={handleVerify}
            loading={verifying}
          >
            验证存证
          </Button>
        </Space>
      </div>

      {verifyResult && (
        <Alert
          message={verifyResult.valid ? '验证通过' : '验证失败'}
          description={
            <div className="space-y-1 mt-2">
              {Object.entries(verifyResult.checks).map(([key, check]) => (
                <div key={key} className="flex items-center gap-2">
                  <span className={check.passed ? 'text-green-500' : 'text-red-500'}>
                    {check.passed ? '✓' : '✗'}
                  </span>
                  <span>{key}: {check.message}</span>
                </div>
              ))}
            </div>
          }
          type={verifyResult.valid ? 'success' : 'error'}
          showIcon
          closable
          onClose={() => setVerifyResult(null)}
        />
      )}

      <Card title="基本信息">
        <Descriptions column={{ xs: 1, sm: 2, md: 2 }} bordered size="small">
          <Descriptions.Item label="存证ID">
            <Space>
              <Text className="font-mono text-sm">{cert.cert_id}</Text>
              <CopyOutlined
                className="text-gray-400 cursor-pointer hover:text-blue-500"
                onClick={() => handleCopy(cert.cert_id)}
              />
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="Trace ID">
            <Space>
              <Text className="font-mono text-sm">{cert.trace_id}</Text>
              <CopyOutlined
                className="text-gray-400 cursor-pointer hover:text-blue-500"
                onClick={() => handleCopy(cert.trace_id)}
              />
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="存证级别">
            <Tag color={levelColors[cert.metadata?.evidence_level || 'L1']}>
              {cert.metadata?.evidence_level || 'N/A'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="事件数量">{cert.event_hashes?.length || 0}</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {cert.metadata?.created_at ? dayjs(cert.metadata.created_at).format('YYYY-MM-DD HH:mm:ss') : 'N/A'}
          </Descriptions.Item>
          <Descriptions.Item label="创建者">{cert.metadata?.created_by || 'N/A'}</Descriptions.Item>
          <Descriptions.Item label="Root Hash" span={2}>
            <Space>
              <Text className="font-mono text-xs" style={{ wordBreak: 'break-all' }}>
                {cert.root_hash}
              </Text>
              <CopyOutlined
                className="text-gray-400 cursor-pointer hover:text-blue-500"
                onClick={() => handleCopy(cert.root_hash)}
              />
            </Space>
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {cert.time_proof && (
        <Card title="时间证明">
          <Descriptions column={{ xs: 1, sm: 2 }} bordered size="small">
            <Descriptions.Item label="证明类型">
              <Tag>{cert.time_proof.proof_type}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="时间戳">
              {dayjs(cert.time_proof.timestamp).format('YYYY-MM-DD HH:mm:ss.SSS')}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {cert.anchor_proof && (
        <Card title="锚定证明">
          <Descriptions column={{ xs: 1, sm: 2 }} bordered size="small">
            <Descriptions.Item label="锚定类型">
              <Tag color={cert.anchor_proof.anchor_type === 'blockchain' ? 'gold' : 'blue'}>
                {cert.anchor_proof.anchor_type}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="锚定ID">{cert.anchor_proof.anchor_id}</Descriptions.Item>
            <Descriptions.Item label="锚定时间">
              {dayjs(cert.anchor_proof.anchor_timestamp).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
            {cert.anchor_proof.blockchain && (
              <>
                <Descriptions.Item label="链ID">
                  {cert.anchor_proof.blockchain.chain_id}
                </Descriptions.Item>
                <Descriptions.Item label="交易哈希">
                  {cert.anchor_proof.blockchain.tx_hash}
                </Descriptions.Item>
                <Descriptions.Item label="区块高度">
                  {cert.anchor_proof.blockchain.block_height}
                </Descriptions.Item>
              </>
            )}
          </Descriptions>
        </Card>
      )}

      <Card title="事件哈希列表">
        <Table
          columns={eventHashColumns}
          dataSource={eventHashData}
          size="small"
          pagination={false}
          scroll={{ y: 300 }}
        />
      </Card>

      <Modal
        title="最小披露证明"
        open={proofModal}
        onCancel={() => setProofModal(false)}
        width={800}
        footer={[
          <Button key="download" icon={<DownloadOutlined />} onClick={handleDownloadProof}>
            下载证明
          </Button>,
          <Button key="close" onClick={() => setProofModal(false)}>
            关闭
          </Button>,
        ]}
      >
        <pre className="bg-gray-50 p-4 rounded-lg overflow-auto text-xs max-h-96">
          {JSON.stringify(proof, null, 2)}
        </pre>
      </Modal>
    </div>
  )
}
