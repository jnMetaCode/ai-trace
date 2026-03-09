package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/ai-trace/server/internal/cert"
)

// ReportType 报告类型
type ReportType string

const (
	ReportTypeAudit      ReportType = "audit"      // 审计报告
	ReportTypeCompliance ReportType = "compliance" // 合规报告
	ReportTypeSummary    ReportType = "summary"    // 摘要报告
)

// ReportFormat 报告格式
type ReportFormat string

const (
	ReportFormatJSON ReportFormat = "json"
	ReportFormatHTML ReportFormat = "html"
	ReportFormatPDF  ReportFormat = "pdf" // 需要额外依赖
)

// ReportRequest 报告请求
type ReportRequest struct {
	Type       ReportType   `json:"type"`
	Format     ReportFormat `json:"format"`
	TraceIDs   []string     `json:"trace_ids,omitempty"`
	CertIDs    []string     `json:"cert_ids,omitempty"`
	StartTime  *time.Time   `json:"start_time,omitempty"`
	EndTime    *time.Time   `json:"end_time,omitempty"`
	TenantID   string       `json:"tenant_id"`
	IncludeRaw bool         `json:"include_raw"`
}

// Report 报告
type Report struct {
	ID          string       `json:"id"`
	Type        ReportType   `json:"type"`
	Format      ReportFormat `json:"format"`
	GeneratedAt time.Time    `json:"generated_at"`
	TenantID    string       `json:"tenant_id"`
	Summary     *Summary     `json:"summary"`
	Details     *Details     `json:"details,omitempty"`
	Content     []byte       `json:"-"` // 生成的内容
	ContentType string       `json:"content_type"`
}

// Summary 摘要统计
type Summary struct {
	TotalTraces       int            `json:"total_traces"`
	TotalEvents       int            `json:"total_events"`
	TotalCertificates int            `json:"total_certificates"`
	EventsByType      map[string]int `json:"events_by_type"`
	CertsByLevel      map[string]int `json:"certs_by_level"`
	TimeRange         *TimeRange     `json:"time_range"`
	TokenUsage        *TokenUsage    `json:"token_usage"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TokenUsage Token 使用统计
type TokenUsage struct {
	TotalPromptTokens     int64 `json:"total_prompt_tokens"`
	TotalCompletionTokens int64 `json:"total_completion_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
}

// Details 详细信息
type Details struct {
	Traces       []TraceDetail  `json:"traces,omitempty"`
	Certificates []CertDetail   `json:"certificates,omitempty"`
	Verifications []Verification `json:"verifications,omitempty"`
}

// TraceDetail 追踪详情
type TraceDetail struct {
	TraceID     string    `json:"trace_id"`
	EventCount  int       `json:"event_count"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Duration    string    `json:"duration"`
	HasCert     bool      `json:"has_cert"`
	CertID      string    `json:"cert_id,omitempty"`
}

// CertDetail 证书详情
type CertDetail struct {
	CertID        string    `json:"cert_id"`
	TraceID       string    `json:"trace_id"`
	RootHash      string    `json:"root_hash"`
	EvidenceLevel string    `json:"evidence_level"`
	CreatedAt     time.Time `json:"created_at"`
	Verified      bool      `json:"verified"`
}

// Verification 验证结果
type Verification struct {
	CertID          string    `json:"cert_id"`
	VerifiedAt      time.Time `json:"verified_at"`
	HashIntegrity   bool      `json:"hash_integrity"`
	SignatureValid  bool      `json:"signature_valid"`
	AnchorVerified  bool      `json:"anchor_verified"`
	OverallValid    bool      `json:"overall_valid"`
}

// Generator 报告生成器
type Generator struct {
	templates map[ReportType]*template.Template
}

// NewGenerator 创建报告生成器
func NewGenerator() (*Generator, error) {
	g := &Generator{
		templates: make(map[ReportType]*template.Template),
	}

	// 解析模板
	auditTmpl, err := template.New("audit").Parse(auditReportTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse audit template: %w", err)
	}
	g.templates[ReportTypeAudit] = auditTmpl

	summaryTmpl, err := template.New("summary").Parse(summaryReportTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse summary template: %w", err)
	}
	g.templates[ReportTypeSummary] = summaryTmpl

	return g, nil
}

// Generate 生成报告
func (g *Generator) Generate(ctx context.Context, req *ReportRequest, data *ReportData) (*Report, error) {
	report := &Report{
		ID:          fmt.Sprintf("rpt_%d", time.Now().UnixNano()),
		Type:        req.Type,
		Format:      req.Format,
		GeneratedAt: time.Now(),
		TenantID:    req.TenantID,
		Summary:     data.Summary,
	}

	if req.IncludeRaw {
		report.Details = data.Details
	}

	// 根据格式生成内容
	switch req.Format {
	case ReportFormatJSON:
		content, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal report: %w", err)
		}
		report.Content = content
		report.ContentType = "application/json"

	case ReportFormatHTML:
		tmpl, ok := g.templates[req.Type]
		if !ok {
			tmpl = g.templates[ReportTypeSummary]
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, report); err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}
		report.Content = buf.Bytes()
		report.ContentType = "text/html"

	case ReportFormatPDF:
		// PDF 生成需要额外依赖，这里先生成 HTML
		return nil, fmt.Errorf("PDF format not yet implemented, use HTML instead")

	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	return report, nil
}

// ReportData 报告数据
type ReportData struct {
	Summary *Summary
	Details *Details
	Certs   []*cert.Certificate
}

// HTML 模板
const auditReportTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>AI-Trace Audit Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; color: #333; }
        .header { border-bottom: 2px solid #2563eb; padding-bottom: 20px; margin-bottom: 30px; }
        .header h1 { color: #1e40af; margin: 0; }
        .header .meta { color: #666; font-size: 14px; margin-top: 10px; }
        .section { margin-bottom: 30px; }
        .section h2 { color: #1e40af; border-bottom: 1px solid #e5e7eb; padding-bottom: 10px; }
        .stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 20px; }
        .stat-card { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 20px; }
        .stat-card .value { font-size: 32px; font-weight: bold; color: #1e40af; }
        .stat-card .label { color: #64748b; font-size: 14px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #e5e7eb; }
        th { background: #f8fafc; font-weight: 600; }
        .badge { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
        .badge-success { background: #dcfce7; color: #166534; }
        .badge-warning { background: #fef3c7; color: #92400e; }
        .footer { margin-top: 40px; padding-top: 20px; border-top: 1px solid #e5e7eb; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>AI-Trace Audit Report</h1>
        <div class="meta">
            Report ID: {{.ID}} | Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}} | Tenant: {{.TenantID}}
        </div>
    </div>

    <div class="section">
        <h2>Summary</h2>
        <div class="stats">
            <div class="stat-card">
                <div class="value">{{.Summary.TotalTraces}}</div>
                <div class="label">Total Traces</div>
            </div>
            <div class="stat-card">
                <div class="value">{{.Summary.TotalEvents}}</div>
                <div class="label">Total Events</div>
            </div>
            <div class="stat-card">
                <div class="value">{{.Summary.TotalCertificates}}</div>
                <div class="label">Certificates</div>
            </div>
            <div class="stat-card">
                <div class="value">{{.Summary.TokenUsage.TotalTokens}}</div>
                <div class="label">Tokens Used</div>
            </div>
        </div>
    </div>

    {{if .Details}}
    <div class="section">
        <h2>Certificate Details</h2>
        <table>
            <thead>
                <tr>
                    <th>Cert ID</th>
                    <th>Trace ID</th>
                    <th>Evidence Level</th>
                    <th>Created At</th>
                    <th>Status</th>
                </tr>
            </thead>
            <tbody>
                {{range .Details.Certificates}}
                <tr>
                    <td><code>{{.CertID}}</code></td>
                    <td><code>{{.TraceID}}</code></td>
                    <td>{{.EvidenceLevel}}</td>
                    <td>{{.CreatedAt.Format "2006-01-02 15:04"}}</td>
                    <td>
                        {{if .Verified}}
                        <span class="badge badge-success">Verified</span>
                        {{else}}
                        <span class="badge badge-warning">Pending</span>
                        {{end}}
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
    {{end}}

    <div class="footer">
        <p>Generated by AI-Trace v0.1 | <a href="https://aitrace.cc">https://aitrace.cc</a></p>
        <p>This report is cryptographically verifiable. Root hashes can be independently verified using the AI-Trace verifier.</p>
    </div>
</body>
</html>`

const summaryReportTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>AI-Trace Summary Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; }
        h1 { color: #1e40af; }
        .summary { background: #f8fafc; padding: 20px; border-radius: 8px; }
    </style>
</head>
<body>
    <h1>AI-Trace Summary</h1>
    <div class="summary">
        <p><strong>Report ID:</strong> {{.ID}}</p>
        <p><strong>Generated:</strong> {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p><strong>Total Traces:</strong> {{.Summary.TotalTraces}}</p>
        <p><strong>Total Events:</strong> {{.Summary.TotalEvents}}</p>
        <p><strong>Total Certificates:</strong> {{.Summary.TotalCertificates}}</p>
    </div>
</body>
</html>`
