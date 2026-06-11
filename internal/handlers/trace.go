package handlers

import (
	"kitchen-trace/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TraceHandler struct {
	db *gorm.DB
}

func NewTraceHandler(db *gorm.DB) *TraceHandler {
	return &TraceHandler{db: db}
}

type MaterialBatchInfo struct {
	MaterialName    string `json:"material_name"`
	Supplier        string `json:"supplier"`
	SupplierBatchNo string `json:"supplier_batch_no"`
	ProductionDate  string `json:"production_date"`
	InboundDate     string `json:"inbound_date"`
	UsedWeight      float64 `json:"used_weight"`
}

type TraceResponse struct {
	TraceCode         string             `json:"trace_code"`
	ProductName       string             `json:"product_name"`
	ProductCode       string             `json:"product_code"`
	ActualQuantity    int                `json:"actual_quantity"`
	WorkerIDs         []string           `json:"worker_ids"`
	InspectionResult  string             `json:"inspection_result"`
	CompletedAt       string             `json:"completed_at"`
	MaterialBatches   []MaterialBatchInfo `json:"material_batches"`
	StoreCode         string             `json:"store_code,omitempty"`
	ReceiptTime       string             `json:"receipt_time,omitempty"`
}

func (h *TraceHandler) QueryByTraceCode(c *gin.Context) {
	traceCode := c.Param("trace_code")
	if traceCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trace_code is required"})
		return
	}

	var wo models.WorkOrder
	if err := h.db.Where("trace_code = ?", traceCode).First(&wo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "trace code not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var usages []models.WorkOrderMaterialUsage
	h.db.Where("work_order_id = ?", wo.ID).Find(&usages)

	materialBatches := make([]MaterialBatchInfo, 0)
	for _, u := range usages {
		var material models.Material
		h.db.First(&material, u.MaterialID)
		materialBatches = append(materialBatches, MaterialBatchInfo{
			MaterialName:    u.MaterialName,
			Supplier:        u.Supplier,
			SupplierBatchNo: u.SupplierBatchNo,
			ProductionDate:  material.ProductionDate.Format("2006-01-02"),
			InboundDate:     material.InboundDate.Format("2006-01-02"),
			UsedWeight:      u.UsedWeight,
		})
	}

	resp := TraceResponse{
		TraceCode:        wo.TraceCode,
		ProductName:      wo.ProductName,
		ProductCode:      wo.ProductCode,
		ActualQuantity:   wo.ActualQuantity,
		WorkerIDs:        wo.WorkerIDs,
		InspectionResult: wo.InspectionResult,
		CompletedAt:      wo.CompletedAt.Format("2006-01-02 15:04:05"),
		MaterialBatches:  materialBatches,
	}

	var receipt models.StoreReceipt
	if err := h.db.Where("trace_code = ?", traceCode).First(&receipt).Error; err == nil {
		resp.StoreCode = receipt.StoreCode
		resp.ReceiptTime = receipt.ReceiptTime.Format("2006-01-02 15:04:05")
	}

	c.JSON(http.StatusOK, resp)
}

type ReverseTraceItem struct {
	TraceCode   string `json:"trace_code"`
	ProductName string `json:"product_name"`
	ProductCode string `json:"product_code"`
	StoreCode   string `json:"store_code,omitempty"`
	ReceiptTime string `json:"receipt_time,omitempty"`
}

func (h *TraceHandler) QueryByMaterialBatch(c *gin.Context) {
	supplierBatchNo := c.Query("supplier_batch_no")
	supplier := c.Query("supplier")

	if supplierBatchNo == "" || supplier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier and supplier_batch_no are required"})
		return
	}

	var usages []models.WorkOrderMaterialUsage
	h.db.Where("supplier = ? AND supplier_batch_no = ?", supplier, supplierBatchNo).Find(&usages)

	if len(usages) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"supplier":          supplier,
			"supplier_batch_no": supplierBatchNo,
			"trace_codes":       []ReverseTraceItem{},
		})
		return
	}

	workOrderIDs := make([]uint64, 0)
	for _, u := range usages {
		workOrderIDs = append(workOrderIDs, u.WorkOrderID)
	}

	var workOrders []models.WorkOrder
	h.db.Where("id IN ? AND status = ?", workOrderIDs, models.WorkOrderStatusDone).Find(&workOrders)

	result := make([]ReverseTraceItem, 0)
	for _, wo := range workOrders {
		item := ReverseTraceItem{
			TraceCode:   wo.TraceCode,
			ProductName: wo.ProductName,
			ProductCode: wo.ProductCode,
		}

		var receipt models.StoreReceipt
		if err := h.db.Where("trace_code = ?", wo.TraceCode).First(&receipt).Error; err == nil {
			item.StoreCode = receipt.StoreCode
			item.ReceiptTime = receipt.ReceiptTime.Format("2006-01-02 15:04:05")
		}

		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"supplier":          supplier,
		"supplier_batch_no": supplierBatchNo,
		"trace_codes":       result,
	})
}
