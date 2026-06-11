package handlers

import (
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
		"message":   "material inbound registered successfully",
		"expiry_date": expiryDate.Format("2006-01-02"),
	})
}
