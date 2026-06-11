package models

import (
	"time"

	"gorm.io/gorm"
)

type MaterialStatus string

const (
	MaterialStatusNormal   MaterialStatus = "normal"
	MaterialStatusNearExp  MaterialStatus = "near_exp"
	MaterialStatusExpired  MaterialStatus = "expired"
)

type WorkOrderStatus string

const (
	WorkOrderStatusPending  WorkOrderStatus = "pending"
	WorkOrderStatusDone     WorkOrderStatus = "done"
)

type Material struct {
	ID              uint64         `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"size:100;not null;index" json:"name"`
	Supplier        string         `gorm:"size:100;not null" json:"supplier"`
	SupplierBatchNo string         `gorm:"size:100;not null" json:"supplier_batch_no"`
	ProductionDate  time.Time      `gorm:"not null" json:"production_date"`
	ShelfLifeDays   int            `gorm:"not null" json:"shelf_life_days"`
	ExpiryDate      time.Time      `gorm:"not null;index" json:"expiry_date"`
	InboundWeight   float64        `gorm:"not null;default:0" json:"inbound_weight"`
	UsedWeight      float64        `gorm:"not null;default:0" json:"used_weight"`
	InboundDate     time.Time      `gorm:"not null;index" json:"inbound_date"`
	Status          MaterialStatus `gorm:"size:20;not null;index" json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type Recipe struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	ProductName string    `gorm:"size:100;not null;uniqueIndex" json:"product_name"`
	ProductCode string    `gorm:"size:50;not null;uniqueIndex" json:"product_code"`
	Items       []RecipeItem `gorm:"foreignKey:RecipeID" json:"items"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RecipeItem struct {
	ID           uint64  `gorm:"primaryKey" json:"id"`
	RecipeID     uint64  `gorm:"not null;index" json:"recipe_id"`
	MaterialName string  `gorm:"size:100;not null" json:"material_name"`
	WeightPerUnit float64 `gorm:"not null" json:"weight_per_unit"`
}

type WorkOrder struct {
	ID            uint64          `gorm:"primaryKey" json:"id"`
	OrderNo       string          `gorm:"size:50;not null;uniqueIndex" json:"order_no"`
	ProductName   string          `gorm:"size:100;not null" json:"product_name"`
	ProductCode   string          `gorm:"size:50;not null" json:"product_code"`
	PlanQuantity  int             `gorm:"not null" json:"plan_quantity"`
	ActualQuantity int            `gorm:"default:0" json:"actual_quantity"`
	RecipeID      uint64          `gorm:"not null" json:"recipe_id"`
	Status        WorkOrderStatus `gorm:"size:20;not null;index" json:"status"`
	WorkerIDs     []string        `gorm:"serializer:json" json:"worker_ids"`
	InspectionResult string       `gorm:"size:500" json:"inspection_result"`
	CompletedAt   *time.Time      `json:"completed_at"`
	TraceCode     string          `gorm:"size:100;uniqueIndex" json:"trace_code"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type WorkOrderMaterialUsage struct {
	ID              uint64  `gorm:"primaryKey" json:"id"`
	WorkOrderID     uint64  `gorm:"not null;index" json:"work_order_id"`
	MaterialID      uint64  `gorm:"not null;index" json:"material_id"`
	MaterialName    string  `gorm:"size:100;not null" json:"material_name"`
	Supplier        string  `gorm:"size:100;not null" json:"supplier"`
	SupplierBatchNo string  `gorm:"size:100;not null" json:"supplier_batch_no"`
	UsedWeight      float64 `gorm:"not null" json:"used_weight"`
	CreatedAt       time.Time `json:"created_at"`
}

type StoreReceipt struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TraceCode   string    `gorm:"size:100;not null;uniqueIndex" json:"trace_code"`
	StoreCode   string    `gorm:"size:50;not null;index" json:"store_code"`
	ReceiptTime time.Time `gorm:"not null" json:"receipt_time"`
	CreatedAt   time.Time `json:"created_at"`
}

func (m *Material) UpdateStatus() {
	now := time.Now()
	daysLeft := int(m.ExpiryDate.Sub(now).Hours() / 24)
	if now.After(m.ExpiryDate) {
		m.Status = MaterialStatusExpired
	} else if daysLeft <= 3 {
		m.Status = MaterialStatusNearExp
	} else {
		m.Status = MaterialStatusNormal
	}
}

func (m *Material) AvailableWeight() float64 {
	return m.InboundWeight - m.UsedWeight
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Material{},
		&Recipe{},
		&RecipeItem{},
		&WorkOrder{},
		&WorkOrderMaterialUsage{},
		&StoreReceipt{},
	)
}
