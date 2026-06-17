package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
)

func AllocateUniquePayAmount(ctx context.Context, orders *repository.OrderRepository, profileID string, baseAmount float64) (float64, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for attempt := 0; attempt < 100; attempt++ {
		paise := rng.Intn(99) + 1
		payAmount := float64(int64(baseAmount*100)+int64(paise)) / 100
		exists, err := orders.PendingPayAmountExists(ctx, profileID, payAmount)
		if err != nil {
			return 0, err
		}
		if !exists {
			return payAmount, nil
		}
	}
	return 0, fmt.Errorf("unable to allocate unique pay amount")
}
