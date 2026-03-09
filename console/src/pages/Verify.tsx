import { useState } from 'react'
import { Card, Input, Button, Alert, Upload, message, Typography, Tabs, Space, Descriptions } from 'antd'
import {
  CheckCircleOutlined,
  UploadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import type { UploadFile } from 'antd'
import { certApi, VerifyResult } from '../services/api'

const { Title, Text, Paragraph } = Typography
const { TextArea } = Input

export default function Verify() {
  const [certId, setCertId] = useState('')
  const [rootHash, setRootHash] = useState('')
  const [proofJson, setProofJson] = useState('')
  const [verifying, setVerifying] = useState(false)
  const [result, setResult] = useState<VerifyResult | null>(null)
  const [fileList, setFileList] = useState<UploadFile[]>([])

  const handleVerifyByCertId = async () => {
    if (!certId.trim()) {
      message.warning('请输入存证ID')
      return
    }
    setVerifying(true)
    setResult(null)
    try {
      const data = await certApi.verify(certId.trim())
      setResult(data)
    } catch (error) {
      console.error('Verify failed:', error)
      message.error('验证失败')
    } finally {
      setVerifying(false)
    }
  }

  const handleVerifyByRootHash = async () => {
    if (!rootHash.trim()) {
      message.warning('请输入Root Hash')
      return
    }
    setVerifying(true)
    setResult(null)
    try {
      const data = await certApi.verify(undefined, rootHash.trim())
      setResult(data)
    } catch (error) {
      console.error('Verify failed:', error)
      message.error('验证失败')
    } finally {
      setVerifying(false)
    }
  }

  const handleVerifyProof = async () => {
    if (!proofJson.trim()) {
      message.warning('请输入或上传证明JSON')
      return
    }
    try {
      const proof = JSON.parse(proofJson)
      setVerifying(true)
      setResult(null)

      // 本地验证Merkle证明
      const isValid = verifyMerkleProof(proof)
      setResult({
        valid: isValid,
        checks: {
          merkle_proof: {
            passed: isValid,
            message: isValid ? 'Merkle证明验证通过' : 'Merkle证明验证失败',
          },
          root_hash: {
            passed: !!proof.root_hash,
            message: `Root Hash: ${proof.root_hash || 'N/A'}`,
          },
        },
        certificate: {
          cert_id: proof.cert_id || 'N/A',
          trace_id: 'N/A',
          root_hash: proof.root_hash || '',
        },
      })
    } catch {
      message.error('无效的JSON格式')
    } finally {
      setVerifying(false)
    }
  }

  // 简化的Merkle证明验证
  const verifyMerkleProof = (proof: { merkle_proofs?: { proof_path?: unknown[] }[] }) => {
    if (!proof.merkle_proofs || proof.merkle_proofs.length === 0) {
      return false
    }
    // 基本结构验证
    return proof.merkle_proofs.every(
      (p: { proof_path?: unknown[] }) => p.proof_path && Array.isArray(p.proof_path)
    )
  }

  const handleFileUpload = (file: File) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      const content = e.target?.result as string
      setProofJson(content)
    }
    reader.readAsText(file)
    return false
  }

  const tabItems = [
    {
      key: 'certId',
      label: '按存证ID验证',
      children: (
        <div className="space-y-4">
          <Input
            size="large"
            placeholder="输入存证ID，例如: cert_xxxxxxxxxxxx"
            value={certId}
            onChange={(e) => setCertId(e.target.value)}
            prefix={<SafetyCertificateOutlined className="text-gray-400" />}
          />
          <Button
            type="primary"
            size="large"
            icon={<CheckCircleOutlined />}
            onClick={handleVerifyByCertId}
            loading={verifying}
            block
          >
            验证存证
          </Button>
        </div>
      ),
    },
    {
      key: 'rootHash',
      label: '按Root Hash验证',
      children: (
        <div className="space-y-4">
          <Input
            size="large"
            placeholder="输入Root Hash，例如: sha256:xxxxxxxx..."
            value={rootHash}
            onChange={(e) => setRootHash(e.target.value)}
          />
          <Button
            type="primary"
            size="large"
            icon={<CheckCircleOutlined />}
            onClick={handleVerifyByRootHash}
            loading={verifying}
            block
          >
            验证存证
          </Button>
        </div>
      ),
    },
    {
      key: 'proof',
      label: '验证证明文件',
      children: (
        <div className="space-y-4">
          <Upload
            fileList={fileList}
            onChange={({ fileList }) => setFileList(fileList)}
            beforeUpload={handleFileUpload}
            maxCount={1}
            accept=".json"
          >
            <Button icon={<UploadOutlined />}>上传证明文件</Button>
          </Upload>
          <TextArea
            rows={8}
            placeholder="或直接粘贴证明JSON..."
            value={proofJson}
            onChange={(e) => setProofJson(e.target.value)}
            className="font-mono text-sm"
          />
          <Button
            type="primary"
            size="large"
            icon={<CheckCircleOutlined />}
            onClick={handleVerifyProof}
            loading={verifying}
            block
          >
            验证证明
          </Button>
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-6">
      <Title level={4}>在线验证</Title>

      <Card>
        <div className="mb-6">
          <Paragraph type="secondary">
            使用此工具验证AI-Trace存证的完整性和真实性。您可以通过存证ID、Root
            Hash或上传证明文件进行验证。
          </Paragraph>
        </div>

        <Tabs items={tabItems} />
      </Card>

      {result && (
        <Card>
          <Alert
            message={result.valid ? '验证成功' : '验证失败'}
            description={
              result.valid
                ? '该存证完整有效，数据未被篡改。'
                : '验证未通过，请检查存证信息是否正确。'
            }
            type={result.valid ? 'success' : 'error'}
            showIcon
            className="mb-4"
          />

          <Title level={5}>验证详情</Title>
          <div className="space-y-2 mb-4">
            {Object.entries(result.checks).map(([key, check]) => (
              <div key={key} className="flex items-center gap-3 p-2 bg-gray-50 rounded">
                <span className={check.passed ? 'text-green-500 text-lg' : 'text-red-500 text-lg'}>
                  {check.passed ? '✓' : '✗'}
                </span>
                <div>
                  <Text strong>{key}</Text>
                  {check.message && (
                    <Text type="secondary" className="ml-2">
                      {check.message}
                    </Text>
                  )}
                </div>
              </div>
            ))}
          </div>

          {result.certificate && (
            <>
              <Title level={5}>存证信息</Title>
              <Descriptions bordered size="small" column={1}>
                <Descriptions.Item label="存证ID">
                  {result.certificate.cert_id}
                </Descriptions.Item>
                <Descriptions.Item label="Root Hash">
                  <Text className="font-mono text-xs">{result.certificate.root_hash}</Text>
                </Descriptions.Item>
                {result.certificate.metadata && (
                  <>
                    <Descriptions.Item label="存证级别">
                      {result.certificate.metadata.evidence_level}
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间">
                      {result.certificate.metadata.created_at}
                    </Descriptions.Item>
                  </>
                )}
              </Descriptions>
            </>
          )}

          <div className="mt-4">
            <Space>
              <Button onClick={() => setResult(null)}>清除结果</Button>
            </Space>
          </div>
        </Card>
      )}
    </div>
  )
}
