package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	lockTTL     = 30 * time.Second
	lockRefresh = 10 * time.Second
	lockPoll    = 100 * time.Millisecond
)

var releaseScript = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`)

var renewScript = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pexpire", KEYS[1], ARGV[2]) else return 0 end`)

type redisLock struct {
	client *redis.Client
	key    string
	token  string
	cancel context.CancelFunc
	done   chan struct{}
}

func acquireLock(ctx context.Context, client *redis.Client, key string) (*redisLock, error) {
	token := uuid.NewString()
	for {
		ok, err := client.SetNX(ctx, key, token, lockTTL).Result()
		if err != nil {
			return nil, err
		}

		if ok {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lockPoll):
		}
	}

	renewCtx, cancel := context.WithCancel(context.Background())
	lock := &redisLock{client: client, key: key, token: token, cancel: cancel, done: make(chan struct{})}
	go lock.keepAlive(renewCtx)
	return lock, nil
}

func (l *redisLock) keepAlive(ctx context.Context) {
	defer close(l.done)
	ticker := time.NewTicker(lockRefresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			renewCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = renewScript.Run(renewCtx, l.client, []string{l.key}, l.token, lockTTL.Milliseconds()).Err()
			cancel()
		}
	}
}

func (l *redisLock) release() {
	l.cancel()
	<-l.done
	releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = releaseScript.Run(releaseCtx, l.client, []string{l.key}, l.token).Err()
}
