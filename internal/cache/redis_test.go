package cache

import "testing"

func TestRedisCache_GetKey(t *testing.T) {
	rc := redisCache{}
	if got := rc.getKey("user"); got != "user" {
		t.Errorf("got %s want %s", got, "user")
	}

	rc.keyPrefix = "config"
	if got := rc.getKey("user"); got != "config:user" {
		t.Errorf("got %s want %s", got, "config:user")
	}
}
