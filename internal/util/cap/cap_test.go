// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package cap

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCapFullFlow(t *testing.T) {
	secret := []byte("a-very-long-secret-key-at-least-16-bytes")
	store := NewMemoryStore(1 * time.Minute)

	manager := NewManager(Config{
		Secret:              secret,
		ChallengeCount:      3, // small count for fast test
		ChallengeSize:       32,
		ChallengeDifficulty: 3, // small difficulty for fast test
		ChallengeTTL:        5 * time.Second,
		TokenTTL:            10 * time.Second,
	}, store)

	scope := "test-scope"
	ctx := context.Background()
	resp, err := manager.Generate(ctx, scope)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp.Challenge.C != 3 {
		t.Errorf("Expected count 3, got %d", resp.Challenge.C)
	}

	// Solve the challenge (acting as client)
	solutions := Solve(resp.Token, resp.Challenge.C, resp.Challenge.S, resp.Challenge.D)

	// Redeem
	redeemResp, err := manager.Redeem(ctx, resp.Token, solutions, scope)
	if err != nil {
		t.Fatalf("Redeem failed: %v", err)
	}
	if !redeemResp.Success {
		t.Fatalf("Redeem returned success=false: %s", redeemResp.Error)
	}
	if redeemResp.Token == "" {
		t.Fatalf("Expected token, got empty")
	}

	// Verify the token
	valid, err := manager.VerifyToken(ctx, redeemResp.Token, scope)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}
	if !valid {
		t.Fatalf("Expected redeem token to be valid")
	}

	// Verify token is one-time use
	validAgain, err := manager.VerifyToken(ctx, redeemResp.Token, scope)
	if err != nil {
		t.Fatalf("VerifyToken second call failed: %v", err)
	}
	if validAgain {
		t.Fatalf("Expected redeem token to be single-use (invalidated after verification)")
	}
}

// TestRedeemConcurrentRace verifies that when N goroutines simultaneously call
// Redeem with the same challenge JWT, exactly one succeeds and the rest are
// rejected with "already_redeemed". This guards against the TOCTOU fix.
func TestRedeemConcurrentRace(t *testing.T) {
	const goroutines = 50

	secret := []byte("race-test-secret-key-at-least-16-bytes")
	store := NewMemoryStore(1 * time.Minute)
	manager := NewManager(Config{
		Secret:              secret,
		ChallengeCount:      1,
		ChallengeSize:       32,
		ChallengeDifficulty: 3,
		ChallengeTTL:        30 * time.Second,
		TokenTTL:            30 * time.Second,
	}, store)

	ctx := context.Background()
	resp, err := manager.Generate(ctx, "login")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	solutions := Solve(resp.Token, resp.Challenge.C, resp.Challenge.S, resp.Challenge.D)

	var (
		wg      sync.WaitGroup
		success atomic.Int32
		barrier = make(chan struct{}) // synchronise goroutine start
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-barrier // wait for the gun
			r, _ := manager.Redeem(ctx, resp.Token, solutions, "login")
			if r != nil && r.Success {
				success.Add(1)
			}
		}()
	}
	close(barrier) // fire all goroutines at once
	wg.Wait()

	if n := success.Load(); n != 1 {
		t.Fatalf("Expected exactly 1 successful Redeem, got %d", n)
	}
}

// TestVerifyTokenConcurrentRace verifies that when N goroutines simultaneously
// call VerifyToken with the same cap token, exactly one succeeds and the rest
// fail. This guards against the GetAndDelete fix.
func TestVerifyTokenConcurrentRace(t *testing.T) {
	const goroutines = 50

	secret := []byte("race-test-secret-key-at-least-16-bytes")
	store := NewMemoryStore(1 * time.Minute)
	manager := NewManager(Config{
		Secret:              secret,
		ChallengeCount:      1,
		ChallengeSize:       32,
		ChallengeDifficulty: 3,
		ChallengeTTL:        30 * time.Second,
		TokenTTL:            30 * time.Second,
	}, store)

	ctx := context.Background()
	resp, _ := manager.Generate(ctx, "login")
	solutions := Solve(resp.Token, resp.Challenge.C, resp.Challenge.S, resp.Challenge.D)
	redeemResp, err := manager.Redeem(ctx, resp.Token, solutions, "login")
	if err != nil || !redeemResp.Success {
		t.Fatalf("Redeem failed: %v %+v", err, redeemResp)
	}
	capToken := redeemResp.Token

	var (
		wg      sync.WaitGroup
		success atomic.Int32
		barrier = make(chan struct{})
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-barrier
			ok, _ := manager.VerifyToken(ctx, capToken, "login")
			if ok {
				success.Add(1)
			}
		}()
	}
	close(barrier)
	wg.Wait()

	if n := success.Load(); n != 1 {
		t.Fatalf("Expected exactly 1 successful VerifyToken, got %d", n)
	}
}
