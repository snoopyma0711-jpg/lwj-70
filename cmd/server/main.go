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

	r := gin.Default()

	api := r.Group("/api/v1")
	{
		materials := api.Group("/materials")
		{
			materials.POST("/inbound", materialHandler.Inbound)
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
	}

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
