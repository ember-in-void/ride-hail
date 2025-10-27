package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"ridehail/internal/shared/config"
	"ridehail/internal/shared/logger"

	adminboot "ridehail/internal/admin/bootstrap"
	driverboot "ridehail/internal/driver/bootstrap"
	rideboot "ridehail/internal/ride/bootstrap"
)

func main() {
	svc := flag.String("service", "ride", "ride|driver|admin|all")
	flag.Parse()

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	go func() { <-quit; cancel() }()

	switch *svc {
	case "ride":
		log := logger.NewLogger("ride-service")
		rideboot.Run(ctx, cfg, log)

	case "driver":
		log := logger.NewLogger("driver-service")
		driverboot.Run(ctx, cfg, log)

	case "admin":
		log := logger.NewLogger("admin-service")
		adminboot.Run(ctx, cfg, log)

	case "all":
		rideLog := logger.NewLogger("ride-service")
		driverLog := logger.NewLogger("driver-service")
		adminLog := logger.NewLogger("admin-service")

		go rideboot.Run(ctx, cfg, rideLog)
		go driverboot.Run(ctx, cfg, driverLog)
		go adminboot.Run(ctx, cfg, adminLog)

	default:
		log := logger.NewLogger("bootstrap")
		log.Fatal(logger.Entry{Action: "invalid_service", Message: *svc})
	}

	<-ctx.Done()
}
