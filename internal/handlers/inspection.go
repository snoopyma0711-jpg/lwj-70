package handlers

import (
	"fmt"
	"kitchen-trace/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InspectionHandler struct {
	db *gorm.DB
}

func NewInspectionHandler(db *gorm.DB) *InspectionHandler {
	return &InspectionHandler{db: db}
}

type CreateInspectionTemplateRequest struct {
	ProductCode   string  `json:"product_code" binding:"required"`
	ProductName   string  `json:"product_name" binding:"required"`
	TemplateName  string  `json:"template_name" binding:"required"`
	ToleranceRate float64 `json:"tolerance_rate" binding:"required,gte=0,lte=1"`
	CheckItems    []struct {
		Name         string              `json:"name" binding:"required"`
		Method       string              `json:"method" binding:"required"`
		StandardType models.StandardType `json:"standard_type" binding:"required,oneof=range match"`
		MinValue     *float64            `json:"min_value,omitempty"`
		MaxValue     *float64            `json:"max_value,omitempty"`
		MatchText    string              `json:"match_text,omitempty"`
		IsKeyPoint   bool                `json:"is_key_point"`
	} `json:"check_items" binding:"required,min=1,dive"`
}

func validateCheckItem(item struct {
	Name         string
	Method       string
	StandardType models.StandardType
	MinValue     *float64
	MaxValue     *float64
	MatchText    string
	IsKeyPoint   bool
}) error {
	if item.StandardType == models.StandardTypeRange {
		if item.MinValue == nil || item.MaxValue == nil {
			return fmt.Errorf("check item '%s': min_value and max_value are required for range type", item.Name)
		}
		if *item.MinValue > *item.MaxValue {
			return fmt.Errorf("check item '%s': min_value cannot be greater than max_value", item.Name)
		}
	} else if item.StandardType == models.StandardTypeMatch {
		if strings.TrimSpace(item.MatchText) == "" {
			return fmt.Errorf("check item '%s': match_text is required for match type", item.Name)
		}
	}
	return nil
}

func (h *InspectionHandler) CreateTemplate(c *gin.Context) {
	var req CreateInspectionTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, item := range req.CheckItems {
		if err := validateCheckItem(struct {
			Name         string
			Method       string
			StandardType models.StandardType
			MinValue     *float64
			MaxValue     *float64
			MatchText    string
			IsKeyPoint   bool
		}(item)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		template := models.InspectionTemplate{
			ProductCode:   req.ProductCode,
			ProductName:   req.ProductName,
			TemplateName:  req.TemplateName,
			ToleranceRate: req.ToleranceRate,
		}
		if err := tx.Create(&template).Error; err != nil {
			return err
		}

		for _, item := range req.CheckItems {
			ci := models.InspectionCheckItem{
				TemplateID:   template.ID,
				Name:         item.Name,
				Method:       item.Method,
				StandardType: item.StandardType,
				MinValue:     item.MinValue,
				MaxValue:     item.MaxValue,
				MatchText:    item.MatchText,
				IsKeyPoint:   item.IsKeyPoint,
			}
			if err := tx.Create(&ci).Error; err != nil {
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
		"message":      "inspection template created successfully",
		"product_code": req.ProductCode,
		"template_name": req.TemplateName,
	})
}

type UpdateInspectionTemplateRequest struct {
	ProductName   *string  `json:"product_name,omitempty"`
	TemplateName  *string  `json:"template_name,omitempty"`
	ToleranceRate *float64 `json:"tolerance_rate,omitempty"`
	CheckItems    *[]struct {
		Name         string              `json:"name" binding:"required"`
		Method       string              `json:"method" binding:"required"`
		StandardType models.StandardType `json:"standard_type" binding:"required,oneof=range match"`
		MinValue     *float64            `json:"min_value,omitempty"`
		MaxValue     *float64            `json:"max_value,omitempty"`
		MatchText    string              `json:"match_text,omitempty"`
		IsKeyPoint   bool                `json:"is_key_point"`
	} `json:"check_items,omitempty"`
}

func (h *InspectionHandler) UpdateTemplate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id"})
		return
	}

	var req UpdateInspectionTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pendingCount int64
	h.db.Model(&models.InspectionTask{}).
		Where("template_id = ? AND status != ?", id, models.InspectionTaskStatusDone).
		Count(&pendingCount)
	if pendingCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot update template while there are pending inspection tasks"})
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var template models.InspectionTemplate
		if err := tx.First(&template, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("template not found")
			}
			return err
		}

		if req.ProductName != nil {
			template.ProductName = *req.ProductName
		}
		if req.TemplateName != nil {
			template.TemplateName = *req.TemplateName
		}
		if req.ToleranceRate != nil {
			if *req.ToleranceRate < 0 || *req.ToleranceRate > 1 {
				return fmt.Errorf("tolerance_rate must be between 0 and 1")
			}
			template.ToleranceRate = *req.ToleranceRate
		}

		if err := tx.Save(&template).Error; err != nil {
			return err
		}

		if req.CheckItems != nil {
			for _, item := range *req.CheckItems {
				if err := validateCheckItem(struct {
					Name         string
					Method       string
					StandardType models.StandardType
					MinValue     *float64
					MaxValue     *float64
					MatchText    string
					IsKeyPoint   bool
				}(item)); err != nil {
					return err
				}
			}

			if err := tx.Where("template_id = ?", id).Delete(&models.InspectionCheckItem{}).Error; err != nil {
				return err
			}

			for _, item := range *req.CheckItems {
				ci := models.InspectionCheckItem{
					TemplateID:   template.ID,
					Name:         item.Name,
					Method:       item.Method,
					StandardType: item.StandardType,
					MinValue:     item.MinValue,
					MaxValue:     item.MaxValue,
					MatchText:    item.MatchText,
					IsKeyPoint:   item.IsKeyPoint,
				}
				if err := tx.Create(&ci).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		if err.Error() == "template not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "inspection template updated successfully"})
}

func (h *InspectionHandler) GetTemplate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id"})
		return
	}

	var template models.InspectionTemplate
	if err := h.db.Preload("CheckItems").First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

func (h *InspectionHandler) ListTemplates(c *gin.Context) {
	var templates []models.InspectionTemplate
	h.db.Preload("CheckItems").Order("created_at DESC").Find(&templates)
	c.JSON(http.StatusOK, templates)
}

func (h *InspectionHandler) DeleteTemplate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id"})
		return
	}

	var pendingCount int64
	h.db.Model(&models.InspectionTask{}).
		Where("template_id = ? AND status != ?", id, models.InspectionTaskStatusDone).
		Count(&pendingCount)
	if pendingCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete template while there are pending inspection tasks"})
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_id = ?", id).Delete(&models.InspectionCheckItem{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&models.InspectionTemplate{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("template not found")
		}
		return nil
	})

	if err != nil {
		if err.Error() == "template not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "inspection template deleted successfully"})
}

func (h *InspectionHandler) PollAndCreateTasks(c *gin.Context) {
	var completedOrders []models.WorkOrder
	h.db.Where("status = ? AND trace_code != ''", models.WorkOrderStatusDone).
		Order("completed_at ASC").
		Find(&completedOrders)

	createdCount := 0
	skippedCount := 0
	noTemplateCount := 0
	errors := make([]string, 0)

	for _, wo := range completedOrders {
		var existingTask models.InspectionTask
		if err := h.db.Where("trace_code = ?", wo.TraceCode).First(&existingTask).Error; err == nil {
			skippedCount++
			continue
		}

		var template models.InspectionTemplate
		if err := h.db.Where("product_code = ?", wo.ProductCode).First(&template).Error; err != nil {
			noTemplateCount++
			continue
		}

		taskNo := fmt.Sprintf("IT-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
		task := models.InspectionTask{
			TaskNo:         taskNo,
			TraceCode:      wo.TraceCode,
			ProductCode:    wo.ProductCode,
			ProductName:    wo.ProductName,
			TemplateID:     template.ID,
			Status:         models.InspectionTaskStatusPending,
			ActualQuantity: wo.ActualQuantity,
			WorkerIDs:      wo.WorkerIDs,
		}

		if err := h.db.Create(&task).Error; err != nil {
			errors = append(errors, fmt.Sprintf("trace_code=%s: %v", wo.TraceCode, err))
			continue
		}
		createdCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "poll completed",
		"created_count":    createdCount,
		"skipped_count":    skippedCount,
		"no_template_count": noTemplateCount,
		"errors":           errors,
	})
}

func (h *InspectionHandler) ListTasks(c *gin.Context) {
	status := c.Query("status")
	productCode := c.Query("product_code")

	query := h.db.Preload("ResultItems").Model(&models.InspectionTask{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if productCode != "" {
		query = query.Where("product_code = ?", productCode)
	}

	var tasks []models.InspectionTask
	query.Order("created_at DESC").Find(&tasks)
	c.JSON(http.StatusOK, tasks)
}

func (h *InspectionHandler) GetTask(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var task models.InspectionTask
	if err := h.db.Preload("ResultItems").First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

type StartInspectionRequest struct {
	InspectorID string `json:"inspector_id" binding:"required"`
}

func (h *InspectionHandler) StartInspection(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req StartInspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	err = h.db.Transaction(func(tx *gorm.DB) error {
		var task models.InspectionTask
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&task, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("task not found")
			}
			return err
		}

		if task.Status == models.InspectionTaskStatusDone {
			return fmt.Errorf("task already completed")
		}
		if task.Status == models.InspectionTaskStatusProcessing {
			return fmt.Errorf("task already in progress")
		}

		task.Status = models.InspectionTaskStatusProcessing
		task.InspectorID = req.InspectorID
		task.StartedAt = &now

		return tx.Save(&task).Error
	})

	if err != nil {
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "inspection started successfully"})
}

type SubmitInspectionItemRequest struct {
	CheckItemID uint64 `json:"check_item_id" binding:"required"`
	ActualValue string `json:"actual_value" binding:"required"`
	FailReason  string `json:"fail_reason,omitempty"`
}

type SubmitInspectionRequest struct {
	Items []SubmitInspectionItemRequest `json:"items" binding:"required,min=1,dive"`
}

func evaluateCheckItem(item models.InspectionCheckItem, actualValue string) (bool, string) {
	actualValue = strings.TrimSpace(actualValue)

	switch item.StandardType {
	case models.StandardTypeRange:
		val, err := strconv.ParseFloat(actualValue, 64)
		if err != nil {
			return false, fmt.Sprintf("无法解析数值: %s", actualValue)
		}
		if item.MinValue != nil && val < *item.MinValue {
			return false, fmt.Sprintf("数值 %.2f 低于最小值 %.2f", val, *item.MinValue)
		}
		if item.MaxValue != nil && val > *item.MaxValue {
			return false, fmt.Sprintf("数值 %.2f 高于最大值 %.2f", val, *item.MaxValue)
		}
		return true, ""
	case models.StandardTypeMatch:
		if strings.Contains(actualValue, item.MatchText) ||
			strings.EqualFold(actualValue, item.MatchText) {
			return true, ""
		}
		return false, fmt.Sprintf("文本 '%s' 不匹配标准 '%s'", actualValue, item.MatchText)
	default:
		return false, "未知的标准类型"
	}
}

func (h *InspectionHandler) SubmitInspection(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req SubmitInspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var task models.InspectionTask
	if err := h.db.First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if task.Status != models.InspectionTaskStatusProcessing {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task must be in processing status"})
		return
	}

	var template models.InspectionTemplate
	if err := h.db.Preload("CheckItems").First(&template, task.TemplateID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "template not found"})
		return
	}

	checkItemMap := make(map[uint64]models.InspectionCheckItem)
	for _, ci := range template.CheckItems {
		checkItemMap[ci.ID] = ci
	}

	for _, item := range req.Items {
		if _, ok := checkItemMap[item.CheckItemID]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid check_item_id: %d", item.CheckItemID),
			})
			return
		}
	}

	submittedIDs := make(map[uint64]bool)
	for _, item := range req.Items {
		if submittedIDs[item.CheckItemID] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("duplicate check_item_id: %d", item.CheckItemID),
			})
			return
		}
		submittedIDs[item.CheckItemID] = true
	}

	for ciID := range checkItemMap {
		if !submittedIDs[ciID] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("missing check_item_id: %d", ciID),
			})
			return
		}
	}

	now := time.Now()
	resultItems := make([]models.InspectionResultItem, 0, len(req.Items))
	keyPointFail := false
	totalItems := 0
	normalFailItems := 0

	for _, item := range req.Items {
		ci := checkItemMap[item.CheckItemID]
		isPass, autoReason := evaluateCheckItem(ci, item.ActualValue)
		failReason := item.FailReason
		if !isPass && failReason == "" {
			failReason = autoReason
		}

		resultItems = append(resultItems, models.InspectionResultItem{
			TaskID:        task.ID,
			CheckItemID:   item.CheckItemID,
			CheckItemName: ci.Name,
			IsKeyPoint:    ci.IsKeyPoint,
			ActualValue:   item.ActualValue,
			IsPass:        isPass,
			FailReason:    failReason,
			CreatedAt:     now,
		})

		totalItems++
		if ci.IsKeyPoint {
			if !isPass {
				keyPointFail = true
			}
		} else {
			if !isPass {
				normalFailItems++
			}
		}
	}

	finalResult := models.InspectionResultPass
	if keyPointFail {
		finalResult = models.InspectionResultFail
	} else {
		normalFailRate := float64(normalFailItems) / float64(totalItems)
		if normalFailRate > template.ToleranceRate {
			finalResult = models.InspectionResultFail
		}
	}

	var report *models.InspectionReport
	var disposal *models.InspectionDisposal

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", task.ID).Delete(&models.InspectionResultItem{}).Error; err != nil {
			return err
		}

		for i := range resultItems {
			if err := tx.Create(&resultItems[i]).Error; err != nil {
				return err
			}
		}

		task.Status = models.InspectionTaskStatusDone
		task.CompletedAt = &now
		task.FinalResult = &finalResult
		if err := tx.Save(&task).Error; err != nil {
			return err
		}

		durationSeconds := 0
		if task.StartedAt != nil {
			durationSeconds = int(now.Sub(*task.StartedAt).Seconds())
		}

		reportItems := make([]models.ReportCheckItem, 0, len(resultItems))
		for _, ri := range resultItems {
			ci := checkItemMap[ri.CheckItemID]
			reportItems = append(reportItems, models.ReportCheckItem{
				CheckItemID:   ri.CheckItemID,
				CheckItemName: ri.CheckItemName,
				IsKeyPoint:    ri.IsKeyPoint,
				Method:        ci.Method,
				StandardType:  ci.StandardType,
				MinValue:      ci.MinValue,
				MaxValue:      ci.MaxValue,
				MatchText:     ci.MatchText,
				ActualValue:   ri.ActualValue,
				IsPass:        ri.IsPass,
				FailReason:    ri.FailReason,
			})
		}

		reportNo := fmt.Sprintf("IR-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
		report = &models.InspectionReport{
			ReportNo:        reportNo,
			TaskID:          task.ID,
			TraceCode:       task.TraceCode,
			ProductCode:     task.ProductCode,
			ProductName:     task.ProductName,
			Items:           reportItems,
			Conclusion:      finalResult,
			InspectorID:     task.InspectorID,
			CompletedAt:     now,
			DurationSeconds: durationSeconds,
			CreatedAt:       now,
		}
		if err := tx.Create(report).Error; err != nil {
			return err
		}

		if finalResult == models.InspectionResultFail {
			defaultMethod := models.DisposalMethodRework
			defaultReason := ""
			failReasons := make([]string, 0)
			for _, ri := range resultItems {
				if !ri.IsPass {
					if ri.FailReason != "" {
						failReasons = append(failReasons, fmt.Sprintf("[%s]%s", ri.CheckItemName, ri.FailReason))
					} else {
						failReasons = append(failReasons, fmt.Sprintf("[%s]不合格", ri.CheckItemName))
					}
				}
			}
			if len(failReasons) > 0 {
				defaultReason = strings.Join(failReasons, "; ")
			}
			if keyPointFail {
				defaultMethod = models.DisposalMethodScrap
			}

			disposalNo := fmt.Sprintf("ID-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
			disposal = &models.InspectionDisposal{
				DisposalNo:     disposalNo,
				ReportID:       report.ID,
				TraceCode:      task.TraceCode,
				ProductCode:    task.ProductCode,
				ProductName:    task.ProductName,
				Method:         defaultMethod,
				Reason:         defaultReason,
				ApplicantID:    task.InspectorID,
				ApprovalStatus: models.ApprovalStatusPending,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := tx.Create(disposal).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"message":      "inspection submitted successfully",
		"task_id":      task.ID,
		"final_result": finalResult,
		"report_id":    report.ID,
		"report_no":    report.ReportNo,
	}
	if disposal != nil {
		response["disposal_id"] = disposal.ID
		response["disposal_no"] = disposal.DisposalNo
	}
	c.JSON(http.StatusOK, response)
}

func (h *InspectionHandler) GetReport(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	var report models.InspectionReport
	if err := h.db.First(&report, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *InspectionHandler) GetReportByTraceCode(c *gin.Context) {
	traceCode := c.Param("trace_code")
	if traceCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trace_code is required"})
		return
	}

	var report models.InspectionReport
	if err := h.db.Where("trace_code = ?", traceCode).First(&report).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *InspectionHandler) ListReports(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	productCode := c.Query("product_code")
	conclusion := c.Query("conclusion")

	query := h.db.Model(&models.InspectionReport{})
	if startDate != "" {
		query = query.Where("completed_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("completed_at <= ?", endDate+" 23:59:59")
	}
	if productCode != "" {
		query = query.Where("product_code = ?", productCode)
	}
	if conclusion != "" {
		query = query.Where("conclusion = ?", conclusion)
	}

	var reports []models.InspectionReport
	query.Order("completed_at DESC").Find(&reports)
	c.JSON(http.StatusOK, reports)
}

type UpdateDisposalRequest struct {
	Method *models.DisposalMethod `json:"method,omitempty"`
	Reason *string                `json:"reason,omitempty"`
}

func (h *InspectionHandler) UpdateDisposal(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid disposal id"})
		return
	}

	var req UpdateDisposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var disposal models.InspectionDisposal
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&disposal, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("disposal not found")
			}
			return err
		}

		if disposal.ApprovalStatus != models.ApprovalStatusPending {
			return fmt.Errorf("cannot update disposal that has been approved/rejected")
		}
		if disposal.Executed {
			return fmt.Errorf("cannot update disposal that has been executed")
		}

		if req.Method != nil {
			switch *req.Method {
			case models.DisposalMethodRework, models.DisposalMethodDegrade, models.DisposalMethodScrap:
				disposal.Method = *req.Method
			default:
				return fmt.Errorf("invalid disposal method")
			}
		}
		if req.Reason != nil {
			disposal.Reason = *req.Reason
		}
		disposal.UpdatedAt = time.Now()

		return tx.Save(&disposal).Error
	})

	if err != nil {
		if err.Error() == "disposal not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "disposal updated successfully"})
}

type ApproveDisposalRequest struct {
	ApproverID string `json:"approver_id" binding:"required"`
	Approved   bool   `json:"approved" binding:"required"`
	Remark     string `json:"remark,omitempty"`
}

func (h *InspectionHandler) ApproveDisposal(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid disposal id"})
		return
	}

	var req ApproveDisposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	err = h.db.Transaction(func(tx *gorm.DB) error {
		var disposal models.InspectionDisposal
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&disposal, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("disposal not found")
			}
			return err
		}

		if disposal.ApprovalStatus != models.ApprovalStatusPending {
			return fmt.Errorf("disposal already processed")
		}

		disposal.ApproverID = req.ApproverID
		disposal.ApprovalRemark = req.Remark
		disposal.ApprovedAt = &now
		disposal.UpdatedAt = now

		if req.Approved {
			disposal.ApprovalStatus = models.ApprovalStatusApproved
		} else {
			disposal.ApprovalStatus = models.ApprovalStatusRejected
		}

		return tx.Save(&disposal).Error
	})

	if err != nil {
		if err.Error() == "disposal not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "disposal approval processed successfully"})
}

func (h *InspectionHandler) ExecuteDisposal(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid disposal id"})
		return
	}

	now := time.Now()
	var statusUpdate string

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var disposal models.InspectionDisposal
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&disposal, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("disposal not found")
			}
			return err
		}

		if disposal.ApprovalStatus != models.ApprovalStatusApproved {
			return fmt.Errorf("disposal must be approved before execution")
		}
		if disposal.Executed {
			return fmt.Errorf("disposal already executed")
		}

		switch disposal.Method {
		case models.DisposalMethodRework:
			statusUpdate = "rework"
		case models.DisposalMethodDegrade:
			statusUpdate = "degraded"
		case models.DisposalMethodScrap:
			statusUpdate = "scrapped"
		default:
			return fmt.Errorf("invalid disposal method")
		}

		var wo models.WorkOrder
		if err := tx.Where("trace_code = ?", disposal.TraceCode).First(&wo).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("work order not found for trace code")
			}
			return err
		}

		inspectionPrefix := ""
		if wo.InspectionResult != "" {
			inspectionPrefix = wo.InspectionResult + "; "
		}
		wo.InspectionResult = fmt.Sprintf("%s处置方式: %s, 状态: %s, 处置单: %s, 执行时间: %s",
			inspectionPrefix, disposal.Method, statusUpdate, disposal.DisposalNo, now.Format("2006-01-02 15:04:05"))

		if err := tx.Save(&wo).Error; err != nil {
			return err
		}

		disposal.Executed = true
		disposal.ExecutedAt = &now
		disposal.UpdatedAt = now

		return tx.Save(&disposal).Error
	})

	if err != nil {
		if err.Error() == "disposal not found" || err.Error() == "work order not found for trace code" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "disposal executed successfully",
		"disposal_method": statusUpdate,
	})
}

func (h *InspectionHandler) ListDisposals(c *gin.Context) {
	approvalStatus := c.Query("approval_status")
	productCode := c.Query("product_code")
	executed := c.Query("executed")

	query := h.db.Model(&models.InspectionDisposal{})
	if approvalStatus != "" {
		query = query.Where("approval_status = ?", approvalStatus)
	}
	if productCode != "" {
		query = query.Where("product_code = ?", productCode)
	}
	if executed != "" {
		executedBool := executed == "true"
		query = query.Where("executed = ?", executedBool)
	}

	var disposals []models.InspectionDisposal
	query.Order("created_at DESC").Find(&disposals)
	c.JSON(http.StatusOK, disposals)
}

func (h *InspectionHandler) GetDisposal(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid disposal id"})
		return
	}

	var disposal models.InspectionDisposal
	if err := h.db.First(&disposal, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "disposal not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, disposal)
}

type StatisticsResponse struct {
	StartTime string                        `json:"start_time"`
	EndTime   string                        `json:"end_time"`
	Products  []ProductInspectionStatistics `json:"products"`
}

type ProductInspectionStatistics struct {
	ProductCode          string            `json:"product_code"`
	ProductName          string            `json:"product_name"`
	TotalTasks           int               `json:"total_tasks"`
	PassCount            int               `json:"pass_count"`
	FailCount            int               `json:"fail_count"`
	PassRate             float64           `json:"pass_rate"`
	AvgDurationSeconds   float64           `json:"avg_duration_seconds"`
	FailReasonDistribution map[string]int   `json:"fail_reason_distribution"`
}

func (h *InspectionHandler) Statistics(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	productCode := c.Query("product_code")

	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}

	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, use YYYY-MM-DD"})
		return
	}
	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, use YYYY-MM-DD"})
		return
	}
	endTime = endTime.AddDate(0, 0, 1).Add(-time.Second)

	reportQuery := h.db.Where("completed_at >= ? AND completed_at <= ?", startTime, endTime)
	if productCode != "" {
		reportQuery = reportQuery.Where("product_code = ?", productCode)
	}

	var reports []models.InspectionReport
	if err := reportQuery.Order("completed_at ASC").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	productMap := make(map[string]*ProductInspectionStatistics)
	for _, report := range reports {
		key := report.ProductCode
		stats, ok := productMap[key]
		if !ok {
			stats = &ProductInspectionStatistics{
				ProductCode:           report.ProductCode,
				ProductName:           report.ProductName,
				TotalTasks:            0,
				PassCount:             0,
				FailCount:             0,
				AvgDurationSeconds:    0,
				FailReasonDistribution: make(map[string]int),
			}
			productMap[key] = stats
		}

		stats.TotalTasks++
		stats.AvgDurationSeconds += float64(report.DurationSeconds)

		if report.Conclusion == models.InspectionResultPass {
			stats.PassCount++
		} else {
			stats.FailCount++
			for _, item := range report.Items {
				if !item.IsPass {
					reason := item.CheckItemName
					if item.FailReason != "" {
						reason = fmt.Sprintf("%s: %s", item.CheckItemName, item.FailReason)
					}
					if len(reason) > 100 {
						reason = reason[:100]
					}
					stats.FailReasonDistribution[reason]++
				}
			}
		}
	}

	result := make([]ProductInspectionStatistics, 0, len(productMap))
	for _, stats := range productMap {
		if stats.TotalTasks > 0 {
			stats.PassRate = float64(stats.PassCount) / float64(stats.TotalTasks)
			stats.AvgDurationSeconds = stats.AvgDurationSeconds / float64(stats.TotalTasks)
		}
		result = append(result, *stats)
	}

	c.JSON(http.StatusOK, StatisticsResponse{
		StartTime: startTime.Format("2006-01-02 15:04:05"),
		EndTime:   endTime.Format("2006-01-02 15:04:05"),
		Products:  result,
	})
}
