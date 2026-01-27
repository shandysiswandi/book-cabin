package main

import (
	"context"
	"time"

	"github.com/shandysiswandi/gobookcabin/internal/app"
)

func main() {
	application := app.New()    // Initialize the application
	wait := application.Start() // Start the application and wait for the termination signal
	<-wait                      // Wait for the application to receive a termination signal

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	application.Stop(ctx) // Stop the application gracefully
}
