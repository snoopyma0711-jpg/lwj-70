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
	LostWeight      float64        `gorm:"not null;default:0" json:"lost_weight"`
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
	return m.InboundWeight - m.UsedWeight - m.LostWeight
}

type MaterialLoss struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	MaterialID      uint64    `gorm:"not null;index" json:"material_id"`
	MaterialName    string    `gorm:"size:100;not null" json:"material_name"`
	Supplier        string    `gorm:"size:100;not null" json:"supplier"`
	SupplierBatchNo string    `gorm:"size:100;not null" json:"supplier_batch_no"`
	LostWeight      float64   `gorm:"not null" json:"lost_weight"`
	Reason          string    `gorm:"size:500;not null" json:"reason"`
	Operator        string    `gorm:"size:100;not null" json:"operator"`
	CreatedAt       time.Time `json:"created_at"`
}

type TraceCodeSeq struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	ProductCode string    `gorm:"size:50;not null;index:idx_product_date,unique" json:"product_code"`
	DateStr     string    `gorm:"size:8;not null;index:idx_product_date,unique" json:"date_str"`
	Seq         int       `gorm:"not null;default:0" json:"seq"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StandardType string

const (
	StandardTypeRange  StandardType = "range"
	StandardTypeMatch  StandardType = "match"
)

type InspectionTaskStatus string

const (
	InspectionTaskStatusPending   InspectionTaskStatus = "pending"
	InspectionTaskStatusProcessing InspectionTaskStatus = "processing"
	InspectionTaskStatusDone      InspectionTaskStatus = "done"
)

type InspectionResult string

const (
	InspectionResultPass InspectionResult = "pass"
	InspectionResultFail InspectionResult = "fail"
)

type DisposalMethod string

const (
	DisposalMethodRework   DisposalMethod = "rework"
	DisposalMethodDegrade  DisposalMethod = "degrade"
	DisposalMethodScrap    DisposalMethod = "scrap"
)

type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

type InspectionTemplate struct {
	ID              uint64              `gorm:"primaryKey" json:"id"`
	ProductCode     string              `gorm:"size:50;not null;uniqueIndex" json:"product_code"`
	ProductName     string              `gorm:"size:100;not null" json:"product_name"`
	TemplateName    string              `gorm:"size:100;not null" json:"template_name"`
	ToleranceRate   float64             `gorm:"not null;default:0.1" json:"tolerance_rate"`
	CheckItems      []InspectionCheckItem `gorm:"foreignKey:TemplateID" json:"check_items"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type InspectionCheckItem struct {
	ID          uint64       `gorm:"primaryKey" json:"id"`
	TemplateID  uint64       `gorm:"not null;index" json:"template_id"`
	Name        string       `gorm:"size:100;not null" json:"name"`
	Method      string       `gorm:"size:500;not null" json:"method"`
	StandardType StandardType `gorm:"size:20;not null" json:"standard_type"`
	MinValue    *float64     `json:"min_value,omitempty"`
	MaxValue    *float64     `json:"max_value,omitempty"`
	MatchText   string       `gorm:"size:200" json:"match_text,omitempty"`
	IsKeyPoint  bool         `gorm:"not null;default:false" json:"is_key_point"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type InspectionTask struct {
	ID              uint64               `gorm:"primaryKey" json:"id"`
	TaskNo          string               `gorm:"size:50;not null;uniqueIndex" json:"task_no"`
	TraceCode       string               `gorm:"size:100;not null;uniqueIndex" json:"trace_code"`
	ProductCode     string               `gorm:"size:50;not null;index" json:"product_code"`
	ProductName     string               `gorm:"size:100;not null" json:"product_name"`
	TemplateID      uint64               `gorm:"not null" json:"template_id"`
	Status          InspectionTaskStatus `gorm:"size:20;not null;index" json:"status"`
	InspectorID     string               `gorm:"size:100" json:"inspector_id,omitempty"`
	ActualQuantity  int                  `gorm:"not null;default:0" json:"actual_quantity"`
	WorkerIDs       []string             `gorm:"serializer:json" json:"worker_ids"`
	StartedAt       *time.Time           `json:"started_at,omitempty"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
	FinalResult     *InspectionResult    `gorm:"size:20" json:"final_result,omitempty"`
	ResultItems     []InspectionResultItem `gorm:"foreignKey:TaskID" json:"result_items,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

type InspectionResultItem struct {
	ID             uint64            `gorm:"primaryKey" json:"id"`
	TaskID         uint64            `gorm:"not null;index" json:"task_id"`
	CheckItemID    uint64            `gorm:"not null" json:"check_item_id"`
	CheckItemName  string            `gorm:"size:100;not null" json:"check_item_name"`
	IsKeyPoint     bool              `gorm:"not null;default:false" json:"is_key_point"`
	ActualValue    string            `gorm:"size:200;not null" json:"actual_value"`
	IsPass         bool              `gorm:"not null" json:"is_pass"`
	FailReason     string            `gorm:"size:500" json:"fail_reason,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

type InspectionReport struct {
	ID              uint64            `gorm:"primaryKey" json:"id"`
	ReportNo        string            `gorm:"size:50;not null;uniqueIndex" json:"report_no"`
	TaskID          uint64            `gorm:"not null;uniqueIndex" json:"task_id"`
	TraceCode       string            `gorm:"size:100;not null;index" json:"trace_code"`
	ProductCode     string            `gorm:"size:50;not null" json:"product_code"`
	ProductName     string            `gorm:"size:100;not null" json:"product_name"`
	Items           []ReportCheckItem `gorm:"serializer:json" json:"items"`
	Conclusion      InspectionResult  `gorm:"size:20;not null" json:"conclusion"`
	InspectorID     string            `gorm:"size:100;not null" json:"inspector_id"`
	CompletedAt     time.Time         `gorm:"not null" json:"completed_at"`
	DurationSeconds int               `gorm:"not null;default:0" json:"duration_seconds"`
	CreatedAt       time.Time         `json:"created_at"`
}

type ReportCheckItem struct {
	CheckItemID   uint64 `json:"check_item_id"`
	CheckItemName string `json:"check_item_name"`
	IsKeyPoint    bool   `json:"is_key_point"`
	Method        string `json:"method"`
	StandardType  StandardType `json:"standard_type"`
	MinValue      *float64 `json:"min_value,omitempty"`
	MaxValue      *float64 `json:"max_value,omitempty"`
	MatchText     string `json:"match_text,omitempty"`
	ActualValue   string `json:"actual_value"`
	IsPass        bool   `json:"is_pass"`
	FailReason    string `json:"fail_reason,omitempty"`
}

type InspectionDisposal struct {
	ID              uint64            `gorm:"primaryKey" json:"id"`
	DisposalNo      string            `gorm:"size:50;not null;uniqueIndex" json:"disposal_no"`
	ReportID        uint64            `gorm:"not null;uniqueIndex" json:"report_id"`
	TraceCode       string            `gorm:"size:100;not null;index" json:"trace_code"`
	ProductCode     string            `gorm:"size:50;not null" json:"product_code"`
	ProductName     string            `gorm:"size:100;not null" json:"product_name"`
	Method          DisposalMethod    `gorm:"size:20;not null" json:"method"`
	Reason          string            `gorm:"size:500;not null" json:"reason"`
	ApplicantID     string            `gorm:"size:100;not null" json:"applicant_id"`
	ApprovalStatus  ApprovalStatus    `gorm:"size:20;not null;default:pending;index" json:"approval_status"`
	ApproverID      string            `gorm:"size:100" json:"approver_id,omitempty"`
	ApprovalRemark  string            `gorm:"size:500" json:"approval_remark,omitempty"`
	ApprovedAt      *time.Time        `json:"approved_at,omitempty"`
	Executed        bool              `gorm:"not null;default:false" json:"executed"`
	ExecutedAt      *time.Time        `json:"executed_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Material{},
		&MaterialLoss{},
		&Recipe{},
		&RecipeItem{},
		&WorkOrder{},
		&WorkOrderMaterialUsage{},
		&StoreReceipt{},
		&TraceCodeSeq{},
		&InspectionTemplate{},
		&InspectionCheckItem{},
		&InspectionTask{},
		&InspectionResultItem{},
		&InspectionReport{},
		&InspectionDisposal{},
	)
}
