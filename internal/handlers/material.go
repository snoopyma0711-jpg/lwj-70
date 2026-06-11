package handlers

import (
	"fmt"
	"kitchen-trace/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MaterialInboundRequest struct {
	Name            string  `json:"name" binding:"required"`
	Supplier        string  `json:"supplier" binding:"required"`
	SupplierBatchNo string  `json:"supplier_batch_no" binding:"required"`
	ProductionDate  string  `json:"production_date" binding:"required"`
	ShelfLifeDays   int     `json:"shelf_life_days" binding:"required,gt=0"`
	Weight          float64 `json:"weight" binding:"required,gt=0"`
}

type MaterialLossRequest struct {
	MaterialID uint64  `json:"material_id" binding:"required,gt=0"`
	LostWeight float64 `json:"lost_weight" binding:"required,gt=0"`
	Reason     string  `json:"reason" binding:"required,max=500"`
	Operator   string  `json:"operator" binding:"required,max=100"`
}

type MaterialHandler struct {
	db *gorm.DB
}

func NewMaterialHandler(db *gorm.DB) *MaterialHandler {
	return &MaterialHandler{db: db}
}

func (h *MaterialHandler) Inbound(c *gin.Context) {
	var req MaterialInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	productionDate, err := time.Parse("2006-01-02", req.ProductionDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid production_date format, use YYYY-MM-DD"})
		return
	}

	expiryDate := productionDate.AddDate(0, 0, req.ShelfLifeDays)
	now := time.Now()

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var existing models.Material
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("supplier = ? AND supplier_batch_no = ?", req.Supplier, req.SupplierBatchNo).
			First(&existing).Error

		if err == nil {
			existing.InboundWeight += req.Weight
			existing.UpdateStatus()
			return tx.Save(&existing).Error
		}

		if err != gorm.ErrRecordNotFound {
			return err
		}

		material := models.Material{
			Name:            req.Name,
			Supplier:        req.Supplier,
			SupplierBatchNo: req.SupplierBatchNo,
			ProductionDate:  productionDate,
			ShelfLifeDays:   req.ShelfLifeDays,
			ExpiryDate:      expiryDate,
			InboundWeight:   req.Weight,
			UsedWeight:      0,
			LostWeight:      0,
			InboundDate:     now,
		}
		material.UpdateStatus()
		return tx.Create(&material).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "material inbound registered successfully",
		"expiry_date": expiryDate.Format("2006-01-02"),
	})
}

func (h *MaterialHandler) ReportLoss(c *gin.Context) {
	var req MaterialLossRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var material models.Material
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("id = ?", req.MaterialID).
			First(&material).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("material not found")
			}
			return err
		}

		available := material.AvailableWeight()
		if req.LostWeight > available {
			return fmt.Errorf("lost weight exceeds available weight: available=%.2f, requested=%.2f", available, req.LostWeight)
		}

		material.LostWeight += req.LostWeight
		if err := tx.Save(&material).Error; err != nil {
			return err
		}

		loss := models.MaterialLoss{
			MaterialID:      material.ID,
			MaterialName:    material.Name,
			Supplier:        material.Supplier,
			SupplierBatchNo: material.SupplierBatchNo,
			LostWeight:      req.LostWeight,
			Reason:          req.Reason,
			Operator:        req.Operator,
			CreatedAt:       time.Now(),
		}
		return tx.Create(&loss).Error
	})

	if err != nil {
		if err.Error() == "material not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "material loss reported successfully",
		"material_id": req.MaterialID,
		"lost_weight": req.LostWeight,
	})
}
