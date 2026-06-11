package handlers

import (
	"fmt"
	"kitchen-trace/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateWorkOrderRequest struct {
	ProductName  string `json:"product_name" binding:"required"`
	PlanQuantity int    `json:"plan_quantity" binding:"required,gt=0"`
}

type MaterialShortage struct {
	MaterialName string  `json:"material_name"`
	Required     float64 `json:"required"`
	Available    float64 `json:"available"`
	Shortage     float64 `json:"shortage"`
}

type WorkOrderHandler struct {
	db *gorm.DB
}

func NewWorkOrderHandler(db *gorm.DB) *WorkOrderHandler {
	return &WorkOrderHandler{db: db}
}

func (h *WorkOrderHandler) CreateWorkOrder(c *gin.Context) {
	var req CreateWorkOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var recipe models.Recipe
	if err := h.db.Where("product_name = ?", req.ProductName).Preload("Items").First(&recipe).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type materialNeed struct {
		Name   string
		Weight float64
	}
	var needs []materialNeed
	for _, item := range recipe.Items {
		needs = append(needs, materialNeed{
			Name:   item.MaterialName,
			Weight: float64(req.PlanQuantity) * item.WeightPerUnit,
		})
	}

	orderNo := fmt.Sprintf("WO-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])

	err := h.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var shortages []MaterialShortage
		type materialUsagePlan struct {
			MaterialName   string
			RequiredWeight float64
			Batches        []batchUsage
		}
		var usages []*materialUsagePlan

		for _, need := range needs {
			var totalAvailable float64
			var batches []models.Material

			err := tx.Set("gorm:query_option", "FOR UPDATE").
				Where("name = ?", need.Name).
				Order("inbound_date ASC, id ASC").
				Find(&batches).Error
			if err != nil {
				return err
			}

			for _, b := range batches {
				if b.ExpiryDate.Before(now) {
					continue
				}
				avail := b.AvailableWeight()
				if avail > 0 {
					totalAvailable += avail
				}
			}

			if totalAvailable < need.Weight {
				shortages = append(shortages, MaterialShortage{
					MaterialName: need.Name,
					Required:     need.Weight,
					Available:    totalAvailable,
					Shortage:     need.Weight - totalAvailable,
				})
				continue
			}

			var plan materialUsagePlan
			plan.MaterialName = need.Name
			plan.RequiredWeight = need.Weight

			remaining := need.Weight
			for _, b := range batches {
				if remaining <= 0 {
					break
				}
				if b.ExpiryDate.Before(now) {
					continue
				}
				avail := b.AvailableWeight()
				if avail <= 0 {
					continue
				}
				useWeight := avail
				if remaining < avail {
					useWeight = remaining
				}
				plan.Batches = append(plan.Batches, batchUsage{
					MaterialID: b.ID,
					Weight:     useWeight,
				})
				remaining -= useWeight
			}
			usages = append(usages, &plan)
		}

		if len(shortages) > 0 {
			return &InsufficientStockError{Shortages: shortages}
		}

		wo := models.WorkOrder{
			OrderNo:      orderNo,
			ProductName:  req.ProductName,
			ProductCode:  recipe.ProductCode,
			PlanQuantity: req.PlanQuantity,
			RecipeID:     recipe.ID,
			Status:       models.WorkOrderStatusPending,
			WorkerIDs:    []string{},
		}
		if err := tx.Create(&wo).Error; err != nil {
			return err
		}

		for _, plan := range usages {
			for _, bu := range plan.Batches {
				var m models.Material
				if err := tx.First(&m, bu.MaterialID).Error; err != nil {
					return err
				}
				avail := m.AvailableWeight()
				if avail < bu.Weight {
					return fmt.Errorf("material %s batch %s insufficient stock, concurrent modification detected", m.Name, m.SupplierBatchNo)
				}
				m.UsedWeight += bu.Weight
				m.UpdateStatus()
				if err := tx.Save(&m).Error; err != nil {
					return err
				}

				usage := models.WorkOrderMaterialUsage{
					WorkOrderID:     wo.ID,
					MaterialID:      m.ID,
					MaterialName:    m.Name,
					Supplier:        m.Supplier,
					SupplierBatchNo: m.SupplierBatchNo,
					UsedWeight:      bu.Weight,
				}
				if err := tx.Create(&usage).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		if ise, ok := err.(*InsufficientStockError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "insufficient material stock",
				"shortages": ise.Shortages,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "work order created successfully",
		"order_no":  orderNo,
		"product":   req.ProductName,
		"quantity":  req.PlanQuantity,
	})
}

type InsufficientStockError struct {
	Shortages []MaterialShortage
}

func (e *InsufficientStockError) Error() string {
	return "insufficient material stock"
}

type batchUsage struct {
	MaterialID uint64
	Weight     float64
}
