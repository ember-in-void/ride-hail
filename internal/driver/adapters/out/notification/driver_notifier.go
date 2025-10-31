package notification

import (
	"context"
	"encoding/json"
	"fmt"

	out "ridehail/internal/driver/application/ports/out"
	"ridehail/internal/shared/logger"
	"ridehail/internal/shared/ws"
)

type driverNotifier struct {
	hub *ws.Hub
	log *logger.Logger
}

func NewDriverNotifier(hub *ws.Hub, log *logger.Logger) out.DriverNotifier {
	return &driverNotifier{hub: hub, log: log}
}

func (n *driverNotifier) SendRideOffer(ctx context.Context, driverID string, offer out.RideOffer) error {
	message := map[string]any{
		"type":        "ride_offer",
		"offer_id":    offer.OfferID,
		"ride_id":     offer.RideID,
		"ride_number": offer.RideNumber,
		"pickup_location": map[string]any{
			"latitude":  offer.PickupLatitude,
			"longitude": offer.PickupLongitude,
			"address":   offer.PickupAddress,
		},
		"destination_location": map[string]any{
			"latitude":  offer.DestinationLatitude,
			"longitude": offer.DestinationLongitude,
			"address":   offer.DestinationAddress,
		},
		"estimated_fare":                  offer.EstimatedFare,
		"driver_earnings":                 offer.DriverEarnings,
		"distance_to_pickup_km":           offer.DistanceToPickupKm,
		"estimated_ride_duration_minutes": offer.EstimatedRideDurationMinutes,
		"expires_at":                      offer.ExpiresAt,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal ride offer: %w", err)
	}

	if err := n.hub.SendToUser(driverID, body); err != nil {
		n.log.Warn(logger.Entry{
			Action:  "send_ride_offer_failed",
			Message: err.Error(),
			Additional: map[string]any{
				"driver_id": driverID,
				"ride_id":   offer.RideID,
			},
		})
		return fmt.Errorf("send ride offer: %w", err)
	}

	n.log.Info(logger.Entry{
		Action:  "ride_offer_sent",
		Message: "ride offer sent to driver",
		RideID:  offer.RideID,
		Additional: map[string]any{
			"driver_id": driverID,
			"offer_id":  offer.OfferID,
		},
	})

	return nil
}

func (n *driverNotifier) SendRideDetails(ctx context.Context, driverID string, details out.RideDetails) error {
	message := map[string]any{
		"type":            "ride_details",
		"ride_id":         details.RideID,
		"passenger_name":  details.PassengerName,
		"passenger_phone": details.PassengerPhone,
		"pickup_location": map[string]any{
			"latitude":  details.PickupLatitude,
			"longitude": details.PickupLongitude,
			"address":   details.PickupAddress,
			"notes":     details.PickupNotes,
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal ride details: %w", err)
	}

	if err := n.hub.SendToUser(driverID, body); err != nil {
		n.log.Warn(logger.Entry{
			Action:  "send_ride_details_failed",
			Message: err.Error(),
			Additional: map[string]any{
				"driver_id": driverID,
				"ride_id":   details.RideID,
			},
		})
		return fmt.Errorf("send ride details: %w", err)
	}

	n.log.Info(logger.Entry{
		Action:  "ride_details_sent",
		Message: "ride details sent to driver",
		RideID:  details.RideID,
		Additional: map[string]any{
			"driver_id": driverID,
		},
	})

	return nil
}

func (n *driverNotifier) IsDriverConnected(driverID string) bool {
	return n.hub.IsUserConnected(driverID)
}
