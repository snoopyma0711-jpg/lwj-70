package handlers

import (
	"fmt"
	"kitchen-trace/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("order_no = ?", orderNo).First(&wo).Error; err != nil {
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

		var seqRecord models.TraceCodeSeq
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("product_code = ? AND date_str = ?", wo.ProductCode, dateStr).
			First(&seqRecord).Error

		if err == gorm.ErrRecordNotFound {
			seqRecord = models.TraceCodeSeq{
				ProductCode: wo.ProductCode,
				DateStr:     dateStr,
				Seq:         1,
			}
			err = tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "product_code"}, {Name: "date_str"}},
				DoUpdates: clause.Assignments(map[string]interface{}{"seq": gorm.Expr("trace_code_seqs.seq + 1")}),
			}).Create(&seqRecord).Error
			if err != nil {
				return err
			}
			if seqRecord.Seq == 0 || seqRecord.ID == 0 {
				if err := tx.Set("gorm:query_option", "FOR UPDATE").
					Where("product_code = ? AND date_str = ?", wo.ProductCode, dateStr).
					First(&seqRecord).Error; err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		} else {
			seqRecord.Seq += 1
			if err := tx.Save(&seqRecord).Error; err != nil {
				return err
			}
		}

		traceCode := fmt.Sprintf("%s-%s-%04d", wo.ProductCode, dateStr, seqRecord.Seq)

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

func (h *WorkOrderHandler) ListWorkOrders(c *gin.Context) {
	var workOrders []models.WorkOrder
	h.db.Order("created_at DESC").Find(&workOrders)
	c.JSON(http.StatusOK, workOrders)
}
