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

func TestParseRedisOpts(t *testing.T) {
	opts, err := parseRedisOpts("redis://redis:6379")
	if err != nil {
		t.FailNow()
	}

	if opts.Username != "default" {
		t.Errorf("wrong username: (got: %s, want: %s)\n", opts.Username, "default")
	}
	if opts.Password != "" {
		t.Errorf("wrong password: (got: %s, want: %s)\n", opts.Password, "")
	}
	if opts.Addr != "redis:6379" {
		t.Errorf("wrong addr: (got: %s, want: %s)\n", opts.Addr, "redis:6379")
	}
	if opts.DB != 0 {
		t.Errorf("wrong DB: (got: %d, want: %d)\n", opts.DB, 0)
	}

	opts, err = parseRedisOpts("redis://default:password@redis:6379/42")
	if err != nil {
		t.FailNow()
	}

	if opts.Username != "default" {
		t.Errorf("wrong username: (got: %s, want: %s)\n", opts.Username, "default")
	}
	if opts.Password != "password" {
		t.Errorf("wrong password: (got: %s, want: %s)\n", opts.Password, "password")
	}
	if opts.Addr != "redis:6379" {
		t.Errorf("wrong addr: (got: %s, want: %s)\n", opts.Addr, "redis:6379")
	}
	if opts.DB != 42 {
		t.Errorf("wrong DB: (got: %d, want: %d)\n", opts.DB, 42)
	}
}
