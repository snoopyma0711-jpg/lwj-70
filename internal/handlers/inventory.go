package handlers

import (
	"kitchen-trace/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InventoryHandler struct {
	db *gorm.DB
}

func NewInventoryHandler(db *gorm.DB) *InventoryHandler {
	return &InventoryHandler{db: db}
}

type BatchDetail struct {
	ID              uint64         `json:"id"`
	Supplier        string         `json:"supplier"`
	SupplierBatchNo string         `json:"supplier_batch_no"`
	ProductionDate  string         `json:"production_date"`
	ExpiryDate      string         `json:"expiry_date"`
	InboundDate     string         `json:"inbound_date"`
	InboundWeight   float64        `json:"inbound_weight"`
	UsedWeight      float64        `json:"used_weight"`
	AvailableWeight float64        `json:"available_weight"`
	Status          models.MaterialStatus `json:"status"`
}

type MaterialInventory struct {
	MaterialName  string        `json:"material_name"`
	TotalAvailable float64       `json:"total_available"`
	WeeklyConsumed float64       `json:"weekly_consumed"`
	Batches        []BatchDetail `json:"batches"`
}

func (h *InventoryHandler) Dashboard(c *gin.Context) {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -7)

	var allMaterials []models.Material
	h.db.Order("name ASC, inbound_date ASC").Find(&allMaterials)

	materialMap := make(map[string]*MaterialInventory)
	for _, m := range allMaterials {
		name := m.Name
		if _, ok := materialMap[name]; !ok {
			materialMap[name] = &MaterialInventory{
				MaterialName:  name,
				TotalAvailable: 0,
				WeeklyConsumed: 0,
				Batches:        []BatchDetail{},
			}
		}

		avail := m.AvailableWeight()
		if m.Status != models.MaterialStatusExpired && avail > 0 {
			materialMap[name].TotalAvailable += avail
		}

		materialMap[name].Batches = append(materialMap[name].Batches, BatchDetail{
			ID:              m.ID,
			Supplier:        m.Supplier,
			SupplierBatchNo: m.SupplierBatchNo,
			ProductionDate:  m.ProductionDate.Format("2006-01-02"),
			ExpiryDate:      m.ExpiryDate.Format("2006-01-02"),
			InboundDate:     m.InboundDate.Format("2006-01-02"),
			InboundWeight:   m.InboundWeight,
			UsedWeight:      m.UsedWeight,
			AvailableWeight: avail,
			Status:          m.Status,
		})
	}

	for name := range materialMap {
		var weeklyConsumed float64
		h.db.Model(&models.WorkOrderMaterialUsage{}).
			Where("material_name = ? AND created_at >= ?", name, weekStart).
			Select("COALESCE(SUM(used_weight), 0)").
			Scan(&weeklyConsumed)
		materialMap[name].WeeklyConsumed = weeklyConsumed
	}

	result := make([]MaterialInventory, 0, len(materialMap))
	for _, v := range materialMap {
		result = append(result, *v)
	}

	c.JSON(http.StatusOK, gin.H{
		"generated_at": now.Format("2006-01-02 15:04:05"),
		"week_start":   weekStart.Format("2006-01-02"),
		"inventory":    result,
	})
}

type RecipeHandler struct {
	db *gorm.DB
}

func NewRecipeHandler(db *gorm.DB) *RecipeHandler {
	return &RecipeHandler{db: db}
}

type CreateRecipeRequest struct {
	ProductName string `json:"product_name" binding:"required"`
	ProductCode string `json:"product_code" binding:"required"`
	Items       []struct {
		MaterialName  string  `json:"material_name" binding:"required"`
		WeightPerUnit float64 `json:"weight_per_unit" binding:"required,gt=0"`
	} `json:"items" binding:"required,min=1"`
}

func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	var req CreateRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		recipe := models.Recipe{
			ProductName: req.ProductName,
			ProductCode: req.ProductCode,
		}
		if err := tx.Create(&recipe).Error; err != nil {
			return err
		}

		for _, item := range req.Items {
			ri := models.RecipeItem{
				RecipeID:      recipe.ID,
				MaterialName:  item.MaterialName,
				WeightPerUnit: item.WeightPerUnit,
			}
			if err := tx.Create(&ri).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "recipe created successfully",
		"product_name": req.ProductName,
		"product_code": req.ProductCode,
	})
}

func (h *RecipeHandler) ListRecipes(c *gin.Context) {
	var recipes []models.Recipe
	h.db.Preload("Items").Find(&recipes)
	c.JSON(http.StatusOK, recipes)
}
