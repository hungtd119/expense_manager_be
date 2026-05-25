package httpapi

import "testing"

func TestAuthRateLimiter(t *testing.T) {
	limiter := newAuthRateLimiter(2)
	if !limiter.Allow("127.0.0.1") || !limiter.Allow("127.0.0.1") {
		t.Fatal("expected first two requests allowed")
	}
	if limiter.Allow("127.0.0.1") {
		t.Fatal("expected third request blocked")
	}
	if !limiter.Allow("127.0.0.2") {
		t.Fatal("expected different IP allowed")
	}
}
