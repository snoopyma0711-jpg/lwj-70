package handlers

import (
	"fmt"
	"kitchen-trace/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StoreReceiptRequest struct {
	TraceCode string `json:"trace_code" binding:"required"`
	StoreCode string `json:"store_code" binding:"required"`
}

type StoreReceiptHandler struct {
	db *gorm.DB
}

func NewStoreReceiptHandler(db *gorm.DB) *StoreReceiptHandler {
	return &StoreReceiptHandler{db: db}
}

func (h *StoreReceiptHandler) Receipt(c *gin.Context) {
	var req StoreReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var wo models.WorkOrder
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("trace_code = ?", req.TraceCode).First(&wo).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("invalid trace code")
			}
			return err
		}

		var existing models.StoreReceipt
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("trace_code = ?", req.TraceCode).First(&existing).Error

		if err == nil {
			if existing.StoreCode != req.StoreCode {
				return fmt.Errorf("trace code already received by store %s, cross-store delivery detected", existing.StoreCode)
			}
			return nil
		}

		if err != gorm.ErrRecordNotFound {
			return err
		}

		receipt := models.StoreReceipt{
			TraceCode:   req.TraceCode,
			StoreCode:   req.StoreCode,
			ReceiptTime: now,
		}
		return tx.Create(&receipt).Error
	})

	if err != nil {
		if err.Error() == "invalid trace code" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "store receipt registered successfully",
		"trace_code":  req.TraceCode,
		"store_code":  req.StoreCode,
		"receipt_time": now.Format("2006-01-02 15:04:05"),
	})
}
