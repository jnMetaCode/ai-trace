package api

import (
	"net/http"

	"github.com/ai-trace/server/internal/anchor"
	"github.com/gin-gonic/gin"
)

// FederatedHandler 联邦节点处理器
type FederatedHandler struct {
	anchor *anchor.FederatedAnchor
}

// RegisterFederatedRoutes 注册联邦节点路由
func (h *Handler) RegisterFederatedRoutes(r *gin.RouterGroup) {
	// 联邦节点 API（公开，供其他节点调用）
	federated := r.Group("/federated")
	{
		// 接收锚定确认请求
		federated.POST("/confirm", h.HandleFederatedConfirm)
		// 验证锚定
		federated.GET("/verify/:anchor_id", h.HandleFederatedVerify)
		// 获取节点信息
		federated.GET("/node/info", h.HandleNodeInfo)
		// 节点发现
		federated.GET("/nodes", h.HandleListNodes)
		// 注册新节点
		federated.POST("/nodes/register", h.HandleRegisterNode)
		// 信任节点管理
		federated.GET("/nodes/trusted", h.HandleListTrustedNodes)
		federated.POST("/nodes/trust", h.HandleTrustNode)
		federated.DELETE("/nodes/trust/:node_id", h.HandleUntrustNode)
	}
}

// HandleFederatedConfirm 处理联邦确认请求
// @Summary 联邦确认请求
// @Description 接收来自其他联邦节点的锚定确认请求
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body anchor.FederatedAnchorRequest true "锚定请求"
// @Success 200 {object} anchor.FederatedAnchorResponse "确认响应"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 503 {object} map[string]interface{} "联邦功能未启用"
// @Router /federated/confirm [post]
func (h *Handler) HandleFederatedConfirm(c *gin.Context) {
	if h.federatedAnchor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Federated anchor not enabled",
		})
		return
	}

	var req anchor.FederatedAnchorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	resp, err := h.federatedAnchor.HandleConfirmRequest(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleFederatedVerify 验证联邦锚定
// @Summary 验证联邦锚定
// @Description 验证指定的联邦锚定是否存在且有效
// @Tags Federation
// @Produce json
// @Param anchor_id path string true "锚定 ID"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 404 {object} map[string]interface{} "锚定不存在"
// @Router /federated/verify/{anchor_id} [get]
func (h *Handler) HandleFederatedVerify(c *gin.Context) {
	if h.federatedAnchor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Federated anchor not enabled",
		})
		return
	}

	anchorID := c.Param("anchor_id")

	// 构造验证请求
	result := &anchor.AnchorResult{
		AnchorID:   anchorID,
		AnchorType: anchor.AnchorTypeFederated,
	}

	valid, err := h.federatedAnchor.Verify(c.Request.Context(), result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if !valid {
		c.JSON(http.StatusNotFound, gin.H{
			"valid": false,
			"error": "Anchor not found or invalid",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":     true,
		"anchor_id": anchorID,
	})
}

// NodeInfoResponse 节点信息响应
type NodeInfoResponse struct {
	NodeID    string   `json:"node_id"`
	PublicKey string   `json:"public_key"`
	Version   string   `json:"version"`
	Endpoints []string `json:"endpoints"`
	Features  []string `json:"features"`
}

// HandleNodeInfo 获取节点信息
// @Summary 获取节点信息
// @Description 获取当前联邦节点的信息
// @Tags Federation
// @Produce json
// @Success 200 {object} NodeInfoResponse "节点信息"
// @Router /federated/node/info [get]
func (h *Handler) HandleNodeInfo(c *gin.Context) {
	if h.federatedAnchor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Federated anchor not enabled",
		})
		return
	}

	features := []string{"anchor", "verify"}

	c.JSON(http.StatusOK, NodeInfoResponse{
		NodeID:    h.federatedAnchor.GetNodeID(),
		PublicKey: h.federatedAnchor.GetPublicKey(),
		Version:   "0.2.0",
		Endpoints: []string{"/federated/confirm", "/federated/verify"},
		Features:  features,
	})
}

// HandleListNodes 列出已知节点
// @Summary 列出联邦节点
// @Description 获取已知的联邦节点列表
// @Tags Federation
// @Produce json
// @Success 200 {object} map[string]interface{} "节点列表"
// @Router /federated/nodes [get]
func (h *Handler) HandleListNodes(c *gin.Context) {
	if h.federatedAnchor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Federated anchor not enabled",
		})
		return
	}

	nodes := h.federatedAnchor.GetKnownNodes()

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// RegisterNodeRequest 注册节点请求
type RegisterNodeRequest struct {
	Endpoint  string `json:"endpoint" binding:"required"`
	PublicKey string `json:"public_key,omitempty"`
}

// HandleRegisterNode 注册新节点
// @Summary 注册联邦节点
// @Description 向本节点注册一个新的联邦节点
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body RegisterNodeRequest true "节点信息"
// @Success 200 {object} map[string]interface{} "注册结果"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Router /federated/nodes/register [post]
func (h *Handler) HandleRegisterNode(c *gin.Context) {
	if h.federatedAnchor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Federated anchor not enabled",
		})
		return
	}

	var req RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	h.federatedAnchor.AddNode(req.Endpoint)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Node registered successfully",
		"endpoint": req.Endpoint,
	})
}

// HandleListTrustedNodes 列出信任的节点
// @Summary 列出信任的联邦节点
// @Description 获取已信任的联邦节点列表
// @Tags Federation
// @Produce json
// @Success 200 {object} map[string]interface{} "信任节点列表"
// @Router /federated/nodes/trusted [get]
func (h *Handler) HandleListTrustedNodes(c *gin.Context) {
	if h.federatedAnchor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Federated anchor not enabled",
		})
		return
	}

	trustedNodes := h.federatedAnchor.GetTrustedNodes()

	c.JSON(http.StatusOK, gin.H{
		"trusted_nodes": trustedNodes,
		"count":         len(trustedNodes),
	})
}

// TrustNodeRequest 信任节点请求
type TrustNodeRequest struct {
	NodeID    string `json:"node_id" binding:"required"`
	PublicKey string `json:"public_key" binding:"required"`
}

// HandleTrustNode 添加信任节点
// @Summary 信任联邦节点
// @Description 将一个联邦节点添加到信任列表，启用签名验证
// @Tags Federation
// @Accept json
// @Produce json
// @Param request body TrustNodeRequest true "节点信息"
// @Success 200 {object} map[string]interface{} "操作结果"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Router /federated/nodes/trust [post]
func (h *Handler) HandleTrustNode(c *gin.Context) {
	if h.federatedAnchor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Federated anchor not enabled",
		})
		return
	}

	var req TrustNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body: node_id and public_key are required",
		})
		return
	}

	// 注册信任的节点
	if err := h.federatedAnchor.RegisterTrustedNode(req.NodeID, req.PublicKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Node trusted successfully",
		"node_id": req.NodeID,
	})
}

// HandleUntrustNode 移除信任节点
// @Summary 取消信任联邦节点
// @Description 从信任列表移除一个联邦节点
// @Tags Federation
// @Produce json
// @Param node_id path string true "节点 ID"
// @Success 200 {object} map[string]interface{} "操作结果"
// @Router /federated/nodes/trust/{node_id} [delete]
func (h *Handler) HandleUntrustNode(c *gin.Context) {
	if h.federatedAnchor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Federated anchor not enabled",
		})
		return
	}

	nodeID := c.Param("node_id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "node_id is required",
		})
		return
	}

	h.federatedAnchor.RemoveTrustedNode(nodeID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Node untrusted successfully",
		"node_id": nodeID,
	})
}
