package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type signupLimiter struct {
	mu       sync.Mutex
	counters map[string]*windowCounter
	limit    int
	window   time.Duration
}

var globalSignupLimiter = &signupLimiter{
	counters: make(map[string]*windowCounter),
	limit:    5,
	window:   time.Hour,
}

func SignupRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		if !globalSignupLimiter.allow(ip) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "too many signup attempts from this IP, try again later",
			})
		}
		return c.Next()
	}
}

func (s *signupLimiter) allow(key string) bool {
	s.mu.Lock()
	counter, ok := s.counters[key]
	if !ok {
		counter = newWindowCounter(s.limit, s.window)
		s.counters[key] = counter
	}
	s.mu.Unlock()
	return counter.allow()
}
