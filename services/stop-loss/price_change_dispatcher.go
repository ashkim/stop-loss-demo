package main

import (
	"context"
	"log"

	"go.temporal.io/sdk/client"
)

func StartPriceChangeDispatcher(temporalClient client.Client, ordersRepo OrdersRepo, pricesChannel <-chan PriceUpdate) {
	for priceUpdate := range pricesChannel {

		// find pending orders for this secuirty:
		workflowIDs, err := ordersRepo.GetPendingWorkflowIDsForSecurity(priceUpdate.Security)
		if err != nil {
			log.Fatal("failed to fetch workflowids from repo", err)
			// TODO: recover from here
		}

		// send workflows the signal that their price has been updated:
		for _, workflowID := range workflowIDs {
			signalData := PriceUpdateSignalData{
				Security: priceUpdate.Security,
				Price:    priceUpdate.Price,
			}

			err := temporalClient.SignalWorkflow(context.Background(), workflowID, "", PriceUpdateSignalName, signalData)
			if err != nil {
				log.Printf("Disaptcher: error signaling workflow %s for security %s: %v", workflowID, priceUpdate.Security, err)
			} else {
				log.Printf("Signaled workflow %s for security %s with price %.2f (via Channel -> Signal)", workflowID, priceUpdate.Security, priceUpdate.Price)
			}
		}
	}
	log.Println("Unexepected price channel closed!")
}
