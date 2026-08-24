package main

import "time"

func newHeartbeatTimer() *time.Timer { return time.NewTimer(50 * time.Millisecond) }
