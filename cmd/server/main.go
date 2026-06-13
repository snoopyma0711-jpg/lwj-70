package main

import (
	"fmt"
	"kitchen-trace/internal/config"
	"kitchen-trace/internal/database"
	"kitchen-trace/internal/handlers"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connected and migrated successfully")

	materialHandler := handlers.NewMaterialHandler(db)
	workOrderHandler := handlers.NewWorkOrderHandler(db)
	traceHandler := handlers.NewTraceHandler(db)
	storeReceiptHandler := handlers.NewStoreReceiptHandler(db)
	inventoryHandler := handlers.NewInventoryHandler(db)
	recipeHandler := handlers.NewRecipeHandler(db)
	inspectionHandler := handlers.NewInspectionHandler(db)

	r := gin.Default()

	api := r.Group("/api/v1")
	{
		materials := api.Group("/materials")
		{
			materials.POST("/inbound", materialHandler.Inbound)
			materials.POST("/loss", materialHandler.ReportLoss)
		}

		recipes := api.Group("/recipes")
		{
			recipes.POST("", recipeHandler.CreateRecipe)
			recipes.GET("", recipeHandler.ListRecipes)
		}

		workorders := api.Group("/workorders")
		{
			workorders.POST("", workOrderHandler.CreateWorkOrder)
			workorders.GET("", workOrderHandler.ListWorkOrders)
			workorders.POST("/:order_no/complete", workOrderHandler.CompleteWorkOrder)
		}

		trace := api.Group("/trace")
		{
			trace.GET("/:trace_code", traceHandler.QueryByTraceCode)
			trace.POST("/batch", traceHandler.BatchQueryByTraceCodes)
			trace.GET("/reverse/material", traceHandler.QueryByMaterialBatch)
		}

		store := api.Group("/store")
		{
			store.POST("/receipt", storeReceiptHandler.Receipt)
		}

		inventory := api.Group("/inventory")
		{
			inventory.GET("/dashboard", inventoryHandler.Dashboard)
		}

		inspectionTemplates := api.Group("/inspection/templates")
		{
			inspectionTemplates.POST("", inspectionHandler.CreateTemplate)
			inspectionTemplates.GET("", inspectionHandler.ListTemplates)
			inspectionTemplates.GET("/:id", inspectionHandler.GetTemplate)
			inspectionTemplates.PUT("/:id", inspectionHandler.UpdateTemplate)
			inspectionTemplates.DELETE("/:id", inspectionHandler.DeleteTemplate)
		}

		inspectionTasks := api.Group("/inspection/tasks")
		{
			inspectionTasks.POST("/poll", inspectionHandler.PollAndCreateTasks)
			inspectionTasks.GET("", inspectionHandler.ListTasks)
			inspectionTasks.GET("/:id", inspectionHandler.GetTask)
			inspectionTasks.POST("/:id/start", inspectionHandler.StartInspection)
			inspectionTasks.POST("/:id/submit", inspectionHandler.SubmitInspection)
		}

		inspectionReports := api.Group("/inspection/reports")
		{
			inspectionReports.GET("", inspectionHandler.ListReports)
			inspectionReports.GET("/:id", inspectionHandler.GetReport)
			inspectionReports.GET("/trace/:trace_code", inspectionHandler.GetReportByTraceCode)
		}

		inspectionDisposals := api.Group("/inspection/disposals")
		{
			inspectionDisposals.GET("", inspectionHandler.ListDisposals)
			inspectionDisposals.GET("/:id", inspectionHandler.GetDisposal)
			inspectionDisposals.PUT("/:id", inspectionHandler.UpdateDisposal)
			inspectionDisposals.POST("/:id/approve", inspectionHandler.ApproveDisposal)
			inspectionDisposals.POST("/:id/execute", inspectionHandler.ExecuteDisposal)
		}

		inspectionStats := api.Group("/inspection")
		{
			inspectionStats.GET("/statistics", inspectionHandler.Statistics)
		}
	}

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
