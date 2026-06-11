package handlers

import (
	"fmt"
	"kitchen-trace/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CompleteWorkOrderRequest struct {
	ActualQuantity  int      `json:"actual_quantity" binding:"required,gt=0"`
	WorkerIDs       []string `json:"worker_ids" binding:"required,min=1"`
	InspectionResult string  `json:"inspection_result" binding:"required"`
}

func (h *WorkOrderHandler) CompleteWorkOrder(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_no is required"})
		return
	}

	var req CompleteWorkOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var wo models.WorkOrder
		if err := tx.Where("order_no = ?", orderNo).First(&wo).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("work order not found")
			}
			return err
		}

		if wo.Status == models.WorkOrderStatusDone {
			return fmt.Errorf("work order already completed")
		}

		now := time.Now()
		dateStr := now.Format("20060102")

		var seq int64
		tx.Model(&models.WorkOrder{}).
			Where("product_code = ? AND trace_code LIKE ?", wo.ProductCode, wo.ProductCode+"-"+dateStr+"%").
			Count(&seq)
		seq++

		traceCode := fmt.Sprintf("%s-%s-%04d", wo.ProductCode, dateStr, seq)

		wo.Status = models.WorkOrderStatusDone
		wo.ActualQuantity = req.ActualQuantity
		wo.WorkerIDs = req.WorkerIDs
		wo.InspectionResult = req.InspectionResult
		wo.CompletedAt = &now
		wo.TraceCode = traceCode

		if err := tx.Save(&wo).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if err.Error() == "work order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	var wo models.WorkOrder
	h.db.Where("order_no = ?", orderNo).First(&wo)

	c.JSON(http.StatusOK, gin.H{
		"message":    "work order completed successfully",
		"trace_code": wo.TraceCode,
		"order_no":   wo.OrderNo,
		"status":     wo.Status,
	})
}
