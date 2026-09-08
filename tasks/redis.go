package tasks

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RedisClient struct {
	mu        sync.Mutex
	conn      net.Conn
	reader    *bufio.Reader
	prefix    string
	ttl       time.Duration
	connected bool
}

func redisNew(host string, port int, password, user, prefix string, ttl int) *RedisClient {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return &RedisClient{connected: false}
	}

	c := &RedisClient{
		conn:      conn,
		reader:    bufio.NewReader(conn),
		prefix:    prefix,
		ttl:       time.Duration(ttl) * time.Minute,
		connected: true,
	}

	if password != "" {
		if user != "" {
			c.cmd("AUTH", user, password)
		} else {
			c.cmd("AUTH", password)
		}
	}
	return c
}

func (c *RedisClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *RedisClient) Set(key, value string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return false
	}
	return c.cmd("SET", c.prefix+key, value, "EX", strconv.Itoa(int(c.ttl.Seconds()))) == nil
}

func (c *RedisClient) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return "", false
	}
	result, err := c.cmdReply("GET", c.prefix+key)
	return result, err == nil && result != ""
}

func (c *RedisClient) Del(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return false
	}
	return c.cmd("DEL", c.prefix+key) == nil
}

// SetNX 设置 key-value，仅当 key 不存在时才设置
// 返回 true 表示设置成功（key 不存在），false 表示 key 已存在
func (c *RedisClient) SetNX(key, value string, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		// Redis 不可用时返回 true，允许继续使用（不去重）
		return true
	}
	seconds := int(ttl.Seconds())
	if seconds <= 0 {
		seconds = int(c.ttl.Seconds())
	}
	err := c.cmd("SET", c.prefix+key, value, "NX", "EX", strconv.Itoa(seconds))
	return err == nil
}

func (c *RedisClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.connected = false
	}
}

func (c *RedisClient) cmd(parts ...string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(parts)))
	for _, p := range parts {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(p), p))
	}
	if _, err := c.conn.Write([]byte(sb.String())); err != nil {
		c.connected = false
		return err
	}
	line, err := c.reader.ReadString('\n')
	if err != nil {
		c.connected = false
		return err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) > 0 && line[0] == '-' {
		return fmt.Errorf("redis: %s", line[1:])
	}
	return nil
}

func (c *RedisClient) cmdReply(parts ...string) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(parts)))
	for _, p := range parts {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(p), p))
	}
	if _, err := c.conn.Write([]byte(sb.String())); err != nil {
		c.connected = false
		return "", err
	}

	line, err := c.reader.ReadString('\n')
	if err != nil {
		c.connected = false
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return "", fmt.Errorf("redis: empty reply")
	}
	switch line[0] {
	case '+', ':':
		return line[1:], nil
	case '-':
		return "", fmt.Errorf("redis: %s", line[1:])
	case '$':
		length, _ := strconv.Atoi(line[1:])
		if length == -1 {
			return "", nil
		}
		buf := make([]byte, length+2)
		c.reader.Read(buf)
		return string(buf[:length]), nil
	default:
		return line, nil
	}
}
